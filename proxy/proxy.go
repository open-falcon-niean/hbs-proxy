package proxy

import (
	"fmt"
	"log"
	"math/rand"

	pfc "github.com/niean/goperfcounter"
	"github.com/open-falcon/common/model"

	"github.com/open-falcon/hbs-proxy/g"
)

var (
	ConnPools    *RpcConnPools
	HbsMap       = make(map[string]string, 0)
	HbsHostnames = make([]string, 0)
	HbsMaxConns  int64
	HbsNum       int
)

func Start() {
	cfg := g.Config()
	if !cfg.Hbs.Enabled {
		log.Println("proxy.Start warning, hbs not enabled")
		return
	}

	initProxy()
	log.Println("proxy.Start ok")
}

// proxy
func ReportStatus(args *model.AgentReportRequest, reply *model.SimpleRpcResponse) error {
	if args.Hostname == "" {
		reply.Code = 1
		return nil
	}

	return proxy("Agent.ReportStatus", args, reply)
}

func MinePlugins(args model.AgentHeartbeatRequest, reply *model.AgentPluginsResponse) error {
	if args.Hostname == "" {
		return nil
	}

	return proxy("Agent.MinePlugins", args, reply)
}

func BuiltinMetrics(args *model.AgentHeartbeatRequest, reply *model.BuiltinMetricResponse) error {
	return proxy("Agent.BuiltinMetrics", args, reply)
}

func TrustableIps(args *model.NullRpcRequest, ips *string) error {
	return proxy("Agent.TrustableIps", args, ips)
}

func proxy(method string, args interface{}, reply interface{}) error {
	// 随机遍历hbs列表，直到数据发送成功 或者 遍历完
	err := fmt.Errorf("proxy connections not available")
	sendOk := false
	rint := rand.Int()
	for i := 0; i < HbsNum && !sendOk; i++ {
		idx := (i + rint) % HbsNum
		host := HbsHostnames[idx]
		addr := HbsMap[host]

		// 过滤掉建连缓慢的host, 否则会严重影响发送速率
		cc := pfc.GetCounterCount(host)
		if cc >= HbsMaxConns {
			continue
		}

		pfc.Counter(host, 1)
		err = ConnPools.Call(addr, method, args, reply)
		pfc.Counter(host, -1)

		if err == nil {
			sendOk = true
		}
	}
	return err
}

// internal
func initProxy() {
	cfg := g.Config()

	// init hbs global configs
	addrs := make([]string, 0)
	for hn, addr := range cfg.Hbs.Cluster {
		HbsHostnames = append(HbsHostnames, hn)
		addrs = append(addrs, addr)
		HbsMap[hn] = addr
	}

	// set consist
	HbsMaxConns = int64(cfg.Hbs.MaxConns)
	HbsNum = len(HbsHostnames)

	// init conn pools
	ConnPools = NewRpcConnPools(8, cfg.Hbs.MaxConns, cfg.Hbs.MaxIdle,
		cfg.Hbs.ConnTimeout, cfg.Hbs.CallTimeout, addrs)
}
