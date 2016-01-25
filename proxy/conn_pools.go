package proxy

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"time"

	spool "github.com/niean/gotools/rpool/conn/simple_conn_pool"
)

// RpcCient, 要实现io.Closer接口
type RpcClient struct {
	cli  *rpc.Client
	name string
}

func (this RpcClient) Name() string {
	return this.name
}

func (this RpcClient) Closed() bool {
	return this.cli == nil
}

func (this RpcClient) Close() error {
	if this.cli != nil {
		err := this.cli.Close()
		this.cli = nil
		return err
	}
	return nil
}

func (this RpcClient) Call(method string, args interface{}, reply interface{}) error {
	return this.cli.Call(method, args, reply)
}

// ConnPools Manager
type RpcConnPools struct {
	sync.RWMutex
	M           map[string]*spool.ConnPool
	MaxPools    int32
	MaxConns    int32
	MaxIdle     int32
	ConnTimeout int32
	CallTimeout int32
}

func NewRpcConnPools(maxPools, maxConns, maxIdle, connTimeout, callTimeout int32, addrs []string) *RpcConnPools {
	cp := &RpcConnPools{
		M:           make(map[string]*spool.ConnPool),
		MaxPools:    maxPools,
		MaxConns:    maxConns,
		MaxIdle:     maxIdle,
		ConnTimeout: connTimeout,
		CallTimeout: callTimeout,
	}
	cp.AddPools(addrs)

	return cp
}

func (this *RpcConnPools) Proc() interface{} {
	this.RLock()
	defer this.RUnlock()
	ret := make(map[string]interface{})
	for key, cp := range this.M {
		ret[key] = cp.Proc()
	}
	return ret
}

func (this *RpcConnPools) Get(address string) (*spool.ConnPool, bool) {
	this.RLock()
	defer this.RUnlock()
	return this.get(address)
}

func (this *RpcConnPools) get(address string) (*spool.ConnPool, bool) {
	p, exists := this.M[address]
	return p, exists
}

func (this *RpcConnPools) AddPools(addrs []string) bool {
	if len(addrs) < 1 {
		return true
	}

	this.Lock()
	defer this.Unlock()

	for _, address := range addrs {
		_, ok := this.addPool(address)
		if !ok {
			return false
		}
	}
	return true
}

func (this *RpcConnPools) AddPool(address string) (*spool.ConnPool, bool) {
	this.Lock()
	defer this.Unlock()
	p, ok := this.addPool(address)
	return p, ok
}

func (this *RpcConnPools) addPool(address string) (*spool.ConnPool, bool) {
	// check size
	if int32(len(this.M)) >= this.MaxPools {
		return nil, false
	}

	// fetch old
	oldp, found := this.M[address]
	if found {
		return oldp, true
	}

	// create one pool
	connTimeout := time.Duration(this.ConnTimeout) * time.Millisecond
	maxConns := this.MaxConns
	maxIdle := this.MaxIdle

	p := spool.NewConnPool(address, address, maxConns, maxIdle)
	p.New = func(connName string) (spool.NConn, error) {
		_, err := net.ResolveTCPAddr("tcp", p.Address)
		if err != nil {
			return nil, err
		}

		conn, err := net.DialTimeout("tcp", p.Address, connTimeout)
		if err != nil {
			return nil, err
		}

		return RpcClient{cli: jsonrpc.NewClient(conn), name: connName}, nil
	}

	// add new pool
	this.M[address] = p

	return p, true
}

func (this *RpcConnPools) RemovePool(addr string) {
	this.Lock()
	defer this.Unlock()
	this.removePool(addr)
}

func (this *RpcConnPools) RemoveAllPools() {
	this.RLock()
	defer this.RUnlock()

	addrs := make([]string, 0, len(this.M))
	for address := range this.M {
		addrs = append(addrs, address)
	}

	if len(addrs) < 1 {
		return
	}
	for _, address := range addrs {
		this.removePool(address)
	}
}

func (this *RpcConnPools) removePool(address string) {
	p, found := this.M[address]
	if !found {
		return
	}

	if p != nil {
		p.Destroy()
	}
	delete(this.M, address)
	return
}

func (this *RpcConnPools) Reset(maxPools, maxConns, maxIdle, connTimeout, callTimeout int32, addrs []string) {
	// rm old pools
	this.RemoveAllPools()

	// reset meta
	this.Lock()
	this.MaxPools = maxPools //TODO: maybe delete me
	this.MaxConns = maxConns
	this.MaxIdle = maxIdle
	this.ConnTimeout = connTimeout
	this.CallTimeout = callTimeout
	this.Unlock()

	// add new pools
	this.AddPools(addrs)
}

// 同步发送, 完成发送或超时后 才能返回
func (this *RpcConnPools) Call(addr, method string, args interface{}, resp interface{}) error {
	connPool, exists := this.Get(addr)
	if !exists {
		// TODO Call的时候新建,有点不合适
		nPool, ok := this.AddPool(addr)
		if !ok {
			return fmt.Errorf("get connection pool fail, addr %s", addr)
		}
		connPool = nPool
	}

	conn, err := connPool.Fetch()
	if err != nil {
		return fmt.Errorf("get connection fail, err %s, proc: %s", err.Error(), connPool.Proc())
	}

	rpcClient := conn.(RpcClient)
	callTimeout := time.Duration(this.CallTimeout) * time.Millisecond

	done := make(chan error)
	go func() {
		done <- rpcClient.Call(method, args, resp)
	}()

	select {
	case <-time.After(callTimeout):
		connPool.ForceClose(conn)
		return fmt.Errorf("call timeout, proc %s", connPool.Proc())
	case err = <-done:
		if err != nil {
			connPool.ForceClose(conn)
			err = fmt.Errorf("call fail, err %s, proc %s", err.Error(), connPool.Proc())
		} else {
			connPool.Release(conn)
		}
		return err
	}
}
