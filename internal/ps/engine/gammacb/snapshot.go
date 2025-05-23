package gammacb

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cubefs/cubefs/depends/tiglabs/raft/proto"
	"github.com/vearch/vearch/v3/internal/pkg/fileutil"
	"github.com/vearch/vearch/v3/internal/pkg/log"
	"github.com/vearch/vearch/v3/internal/proto/vearchpb"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	// trasport 10M everytime
	buf_size = 1024000 * 10
)

var _ proto.Snapshot = &GammaSnapshot{}

type GammaSnapshot struct {
	sn           int64
	index        int64
	path         string
	infos        []fs.DirEntry
	absFileNames []string
	reader       *os.File
	size         int64
}

func (g *GammaSnapshot) Next() ([]byte, error) {
	var err error

	if int(g.index) >= len(g.absFileNames) && g.size == 0 {
		log.Debug("leader send over, leader finish snapshot.")
		snapShotMsg := &vearchpb.SnapshotMsg{
			Status: vearchpb.SnapshotStatus_Finish,
		}
		data, err := protobuf.Marshal(snapShotMsg)
		if err != nil {
			return data, err
		} else {
			return data, io.EOF
		}
	}
	if g.reader == nil {
		filePath := g.absFileNames[g.index]
		g.index = g.index + 1
		log.Debug("g.index is [%+v] ", g.index)
		log.Debug("g.absFileNames length is [%+v] ", len(g.absFileNames))
		info, _ := os.Stat(filePath)
		if info.IsDir() {
			log.Debug("dir:[%s] name:[%s] is dir , so skip sync", g.path, info.Name())
			snapShotMsg := &vearchpb.SnapshotMsg{
				Status: vearchpb.SnapshotStatus_Running,
			}
			return protobuf.Marshal(snapShotMsg)
		}
		g.size = info.Size()
		reader, err := os.Open(filePath)
		log.Debug("next reader info [%+v],path [%s],name [%s]", info, g.path, filePath)
		if err != nil {
			return nil, err
		}
		g.reader = reader
	}

	byteData := make([]byte, int64(math.Min(buf_size, float64(g.size))))
	size, err := g.reader.Read(byteData)
	if err != nil {
		return nil, err
	}
	g.size = g.size - int64(size)
	log.Debug("current g.size [%+v], info size [%+v]", g.size, size)
	if g.size == 0 {
		if err := g.reader.Close(); err != nil {
			return nil, err
		}
		g.reader = nil
	}
	// snapshot proto msg
	snapShotMsg := &vearchpb.SnapshotMsg{
		FileName: g.absFileNames[g.index-1],
		Data:     byteData,
		Status:   vearchpb.SnapshotStatus_Running,
	}
	return protobuf.Marshal(snapShotMsg)
}

func (g *GammaSnapshot) ApplyIndex() uint64 {
	return uint64(g.sn)
}

func (g *GammaSnapshot) Close() {
	if g.reader != nil {
		_ = g.reader.Close()
	}
}

func (ge *gammaEngine) NewSnapshot() (proto.Snapshot, error) {
	ge.lock.RLock()
	defer ge.lock.RUnlock()
	err := ge.BackupSpace("create")
	if err != nil {
		log.Error("create snapshot error:[%v]", err)
		return nil, err
	}

	backupPath := filepath.Join(ge.path, "backup")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		log.Error("backup path [%s] not exist, err:[%v]", backupPath, err)
		return nil, err
	}

	baseSNFile := filepath.Join(ge.path, indexSn)
	baseSNValue := []byte("0")
	if b, err := os.ReadFile(baseSNFile); err == nil {
		baseSNValue = b
	}

	backupSNFile := filepath.Join(backupPath, indexSn)
	if _, err := os.Stat(backupSNFile); os.IsNotExist(err) {
		if err := os.WriteFile(backupSNFile, baseSNValue, 0644); err != nil {
			log.Error("failed to create sn file in backup: %v", err)
			return nil, err
		}
	}

	infos, err := os.ReadDir(backupPath)
	if err != nil {
		log.Error("failed to read backup directory: %v", err)
		return nil, err
	}

	absFileNames, err := fileutil.GetAllFileNames(backupPath)
	if err != nil {
		log.Error("failed to get file names: %v", err)
		return nil, err
	}

	b, err := os.ReadFile(backupSNFile)
	if err != nil {
		b = []byte("0")
	}

	sn, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return nil, err
	}
	if sn < 0 {
		return nil, vearchpb.NewError(vearchpb.ErrorEnum_INTERNAL_ERROR, fmt.Errorf("read sn:[%d] less than zero", sn))
	}
	return &GammaSnapshot{path: ge.path, sn: sn, infos: infos, absFileNames: absFileNames}, nil
}

func (ge *gammaEngine) ApplySnapshot(peers []proto.Peer, iter proto.SnapIterator) error {
	var out *os.File

	for {
		bs, err := iter.Next()
		if err != nil && err != io.EOF {
			return err
		}
		if bs == nil {
			continue
		}

		msg := &vearchpb.SnapshotMsg{}
		err = protobuf.Unmarshal(bs, msg)
		if err != nil {
			return err
		}
		if msg.Status == vearchpb.SnapshotStatus_Finish {
			if out != nil {
				if err := out.Close(); err != nil {
					return err
				}
				out = nil
			}
			log.Debug("follower receive finish.")
			break
		}
		if msg.Data == nil || len(msg.Data) == 0 {
			log.Debug("msg data is nil.")
			continue
		}
		if out != nil {
			if err := out.Close(); err != nil {
				return err
			}
			out = nil
		}
		// create dir
		fileDir := filepath.Dir(msg.FileName)
		_, exist := os.Stat(fileDir)
		if os.IsNotExist(exist) {
			log.Debug("create dir [%+v]", fileDir)
			err := os.MkdirAll(fileDir, os.ModePerm)
			if err != nil {
				return err
			}
		}
		// create file, append write mode
		if out, err = os.OpenFile(msg.FileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660); err != nil {
			return err
		}
		log.Debug("write file path [%s], name [%s], size [%d]", ge.path, msg.FileName, len(msg.Data))
		if _, err = out.Write(msg.Data); err != nil {
			return err
		}
	}

	backupPath := filepath.Join(ge.path, "backup")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory doesn't exist: %v", err)
	}

	files, err := os.ReadDir(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	for _, file := range files {
		srcPath := filepath.Join(backupPath, file.Name())
		dstPath := filepath.Join(ge.path, file.Name())

		if _, err := os.Stat(dstPath); err == nil {
			if err := os.RemoveAll(dstPath); err != nil {
				return fmt.Errorf("failed to remove existing file %s: %v", dstPath, err)
			}
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to move file %s to %s: %v", srcPath, dstPath, err)
		}
		log.Debug("Moved %s to %s", srcPath, dstPath)
	}

	if err := os.RemoveAll(backupPath); err != nil {
		log.Warn("Failed to remove backup directory after successful move: %v", err)
	}

	return nil
}
