// Copyright 2019 The Vearch Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/smallnest/pool"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/share"
	"github.com/spf13/cast"
	"github.com/vearch/vearch/v3/internal/config"
	"github.com/vearch/vearch/v3/internal/entity"
	"github.com/vearch/vearch/v3/internal/pkg/log"
	"github.com/vearch/vearch/v3/internal/proto/vearchpb"
)

var defaultConcurrentNum int = 2000

type RpcClient struct {
	serverAddress []string
	clientPool    *pool.Pool
	concurrent    chan bool
	concurrentNum int
}

func NewRpcClient(serverAddress ...string) (*RpcClient, error) {
	var d client.ServiceDiscovery
	if len(serverAddress) == 1 {
		d, _ = client.NewPeer2PeerDiscovery("tcp@"+serverAddress[0], "")
	} else {
		arr := make([]*client.KVPair, len(serverAddress))
		for i, addr := range serverAddress {
			arr[i] = &client.KVPair{Key: addr}
		}
		d, _ = client.NewMultipleServersDiscovery(arr)
	}

	clientPool := &pool.Pool{New: func() interface{} {
		oneclient := client.NewOneClient(client.Failfast, client.RandomSelect, d, ClientOption)
		return oneclient
	}}

	r := &RpcClient{serverAddress: serverAddress, clientPool: clientPool}
	r.concurrentNum = defaultConcurrentNum
	if config.Conf().Router.ConcurrentNum > 0 {
		r.concurrentNum = config.Conf().Router.ConcurrentNum
	}
	r.concurrent = make(chan bool, r.concurrentNum)
	return r, nil
}

func (r *RpcClient) Close() error {
	var e error
	r.clientPool.Range(func(v interface{}) bool {
		if err := v.(*client.OneClient).Close(); err != nil {
			log.Error("close client has err:[%s]", err.Error())
			e = err
		}
		return true
	})
	return e
}

func (r *RpcClient) Execute(ctx context.Context, servicePath string, args interface{}, reply *vearchpb.PartitionData) (err error) {
	r.concurrent <- true
	defer func() {
		<-r.concurrent
		if r := recover(); r != nil {
			err = errors.New(cast.ToString(r))
			log.Error(err.Error())
		}
	}()
	select {
	case <-ctx.Done():
		msg := fmt.Sprintf("Too much concurrency causes time out, the max num of concurrency is [%d]", r.concurrentNum)
		err = vearchpb.NewError(vearchpb.ErrorEnum_TIMEOUT, errors.New(msg))
		log.Errorf(msg)
		return
	default:
		var (
			md map[string]string
			ok bool
		)
		if m := ctx.Value(share.ReqMetaDataKey); m != nil {
			md, ok = m.(map[string]string)
			if !ok {
				md = make(map[string]string)
			}
		} else {
			md = make(map[string]string)
		}
		if endTime, ok := ctx.Value(entity.RPC_TIME_OUT).(time.Time); ok {
			timeout := int64((time.Until(endTime) + time.Millisecond - 1) / time.Millisecond)
			if timeout < 1 {
				msg := fmt.Sprintf("timeout[%d] is too small", timeout)
				err = vearchpb.NewError(vearchpb.ErrorEnum_PARAM_ERROR, errors.New(msg))
				log.Errorf(msg)
				return
			}
			md[string(entity.RPC_TIME_OUT)] = strconv.FormatInt(int64(timeout), 10)
		}
		if span := opentracing.SpanFromContext(ctx); span != nil {
			span.Tracer().Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(md))
		}
		ctx = context.WithValue(ctx, share.ReqMetaDataKey, md)
		cli := r.clientPool.Get().(*client.OneClient)
		defer r.clientPool.Put(cli)
		if err = cli.Call(ctx, servicePath, serviceMethod, args, reply); err != nil {
			log.Error("call %s err: %v", servicePath+serviceMethod, err.Error())
			if errors.Is(err, context.DeadlineExceeded) {
				err = vearchpb.NewError(vearchpb.ErrorEnum_TIMEOUT, nil)
			} else {
				err = vearchpb.NewError(vearchpb.ErrorEnum_CALL_RPCCLIENT_FAILED, err)
			}
		}
		return
	}
}

func (r *RpcClient) GetAddress(i int) string {
	if r == nil || len(r.serverAddress) <= i {
		return ""
	}
	if i < 0 {
		return strings.Join(r.serverAddress, ",")
	}
	return r.serverAddress[i]
}

func (r *RpcClient) GetConcurrent() int {
	return len(r.concurrent)
}
