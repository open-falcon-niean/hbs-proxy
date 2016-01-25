package rpc

import (
	"github.com/open-falcon/common/model"
	"github.com/open-falcon/hbs-proxy/proxy"
)

type Agent int

func (t *Agent) ReportStatus(args *model.AgentReportRequest, reply *model.SimpleRpcResponse) error {
	return proxy.ReportStatus(args, reply)
}

func (t *Agent) MinePlugins(args model.AgentHeartbeatRequest, reply *model.AgentPluginsResponse) error {
	return proxy.MinePlugins(args, reply)
}

func (t *Agent) BuiltinMetrics(args *model.AgentHeartbeatRequest, reply *model.BuiltinMetricResponse) error {
	return proxy.BuiltinMetrics(args, reply)
}

func (t *Agent) TrustableIps(args *model.NullRpcRequest, ips *string) error {
	return proxy.TrustableIps(args, ips)
}
