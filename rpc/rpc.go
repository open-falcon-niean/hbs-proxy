package rpc

import (
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/open-falcon/hbs-proxy/g"
)

func Start() {
	cfg := g.Config()
	if !cfg.Rpc.Enabled {
		log.Println("rpc.Start warning, not enable")
		return
	}
	addr := cfg.Rpc.Listen

	server := rpc.NewServer()
	server.Register(new(Agent))

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatalln("rpc.Start error", e)
	} else {
		log.Println("rpc.Start ok, listening on", addr)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if cfg.Debug {
					log.Println("rpc accept fail:", err)
				}
				time.Sleep(time.Duration(100) * time.Millisecond)
				continue
			}
			go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()
}
