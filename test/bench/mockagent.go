package main

import (
	"flag"
	"fmt"
	"github.com/toolkits/net"
	"log"
	"math"
	"net/rpc"
	"sync"
	"time"

	"github.com/open-falcon/common/model"
)

const (
	hbsAddr = "127.0.0.1:6030"
	timeout = time.Duration(10) * time.Second
)

var (
	hbsClient = &SingleConnRpcClient{RpcServer: hbsAddr, Timeout: timeout}
	rpcs      = map[string]func(){"TrustableIps": trustable_ips, "BuiltinMetrics": builtin_metrics, "MinePlugins": mine_plugins, "ReportStatus": report_status}
)

func main() {
	// flag
	method := flag.String("m", "TrustableIps", "ticker interval")
	flag.Parse()

	if f, ok := rpcs[*method]; ok {
		fmt.Println("[mockagent] method:", *method)
		f()
		return
	} else {
		fmt.Println("[mockagent] all methods")
		for _, f := range rpcs {
			f()
			time.Sleep(time.Second)
		}
	}

	fmt.Println("[mockagent] done")
}

// rpc
func trustable_ips() {
	var ips string
	err := hbsClient.Call("Agent.TrustableIps", model.NullRpcRequest{}, &ips)
	if err != nil {
		fmt.Println("[mockagent] TrustableIps error:", err)
	} else {
		fmt.Println("[mockagent] TrustableIps:", ips)
	}
}

func builtin_metrics() {
	hostname := "work"
	req := model.AgentHeartbeatRequest{
		Hostname: hostname,
		Checksum: "abc123",
	}

	var resp model.BuiltinMetricResponse
	err := hbsClient.Call("Agent.BuiltinMetrics", req, &resp)
	if err != nil {
		fmt.Println("[mockagent] BuiltinMetrics error:", err)
	} else {
		fmt.Println("[mockagent] BuiltinMetrics:", resp)
	}
}

func mine_plugins() {
	hostname := "work"
	req := model.AgentHeartbeatRequest{
		Hostname: hostname,
	}

	var resp model.AgentPluginsResponse
	err := hbsClient.Call("Agent.MinePlugins", req, &resp)
	if err != nil {
		fmt.Println("[mockagent] MinePlugins error:", err)
	} else {
		fmt.Println("[mockagent] MinePlugins:", resp)
	}
}

func report_status() {
	hostname := "work"
	req := model.AgentReportRequest{
		Hostname:      hostname,
		IP:            "128.1.1.2",
		AgentVersion:  "v.test",
		PluginVersion: "/home/to/plugin",
	}

	var resp model.SimpleRpcResponse
	err := hbsClient.Call("Agent.ReportStatus", req, &resp)

	if err != nil || resp.Code != 0 {
		fmt.Println("[mockagent] ReportStatus error:", err, resp)
	} else {
		fmt.Println("[mockagent] ReportStatus:", resp)
	}
}

// connection
type SingleConnRpcClient struct {
	sync.Mutex
	rpcClient *rpc.Client
	RpcServer string
	Timeout   time.Duration
}

func (this *SingleConnRpcClient) close() {
	if this.rpcClient != nil {
		this.rpcClient.Close()
		this.rpcClient = nil
	}
}

func (this *SingleConnRpcClient) insureConn() {
	if this.rpcClient != nil {
		return
	}

	var err error
	var retry int = 1

	for {
		if this.rpcClient != nil {
			return
		}

		this.rpcClient, err = net.JsonRpcClient("tcp", this.RpcServer, this.Timeout)
		if err == nil {
			return
		}

		log.Printf("dial %s fail: %v", this.RpcServer, err)

		if retry > 6 {
			retry = 1
		}

		time.Sleep(time.Duration(math.Pow(2.0, float64(retry))) * time.Second)

		retry++
	}
}

func (this *SingleConnRpcClient) Call(method string, args interface{}, reply interface{}) error {

	this.Lock()
	defer this.Unlock()

	this.insureConn()

	timeout := time.Duration(50 * time.Second)
	done := make(chan error)

	go func() {
		err := this.rpcClient.Call(method, args, reply)
		done <- err
	}()

	select {
	case <-time.After(timeout):
		log.Printf("[WARN] rpc call timeout %v => %v", this.rpcClient, this.RpcServer)
		this.close()
	case err := <-done:
		if err != nil {
			this.close()
			return err
		}
	}

	return nil
}
