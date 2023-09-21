// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package gamma_api

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type MemoryInfo struct {
	_tab flatbuffers.Table
}

func GetRootAsMemoryInfo(buf []byte, offset flatbuffers.UOffsetT) *MemoryInfo {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &MemoryInfo{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *MemoryInfo) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *MemoryInfo) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *MemoryInfo) TableMem() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemoryInfo) MutateTableMem(n int64) bool {
	return rcv._tab.MutateInt64Slot(4, n)
}

func (rcv *MemoryInfo) IndexMem() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemoryInfo) MutateIndexMem(n int64) bool {
	return rcv._tab.MutateInt64Slot(6, n)
}

func (rcv *MemoryInfo) VectorMem() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemoryInfo) MutateVectorMem(n int64) bool {
	return rcv._tab.MutateInt64Slot(8, n)
}

func (rcv *MemoryInfo) FieldRangeMem() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemoryInfo) MutateFieldRangeMem(n int64) bool {
	return rcv._tab.MutateInt64Slot(10, n)
}

func (rcv *MemoryInfo) BitmapMem() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemoryInfo) MutateBitmapMem(n int64) bool {
	return rcv._tab.MutateInt64Slot(12, n)
}

func MemoryInfoStart(builder *flatbuffers.Builder) {
	builder.StartObject(5)
}
func MemoryInfoAddTableMem(builder *flatbuffers.Builder, tableMem int64) {
	builder.PrependInt64Slot(0, tableMem, 0)
}
func MemoryInfoAddIndexMem(builder *flatbuffers.Builder, indexMem int64) {
	builder.PrependInt64Slot(1, indexMem, 0)
}
func MemoryInfoAddVectorMem(builder *flatbuffers.Builder, vectorMem int64) {
	builder.PrependInt64Slot(2, vectorMem, 0)
}
func MemoryInfoAddFieldRangeMem(builder *flatbuffers.Builder, fieldRangeMem int64) {
	builder.PrependInt64Slot(3, fieldRangeMem, 0)
}
func MemoryInfoAddBitmapMem(builder *flatbuffers.Builder, bitmapMem int64) {
	builder.PrependInt64Slot(4, bitmapMem, 0)
}
func MemoryInfoEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}