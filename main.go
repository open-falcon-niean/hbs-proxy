package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/open-falcon/hbs-proxy/g"
	"github.com/open-falcon/hbs-proxy/http"
	"github.com/open-falcon/hbs-proxy/proxy"
	"github.com/open-falcon/hbs-proxy/rpc"
)

func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	// config
	g.ParseConfig(*cfg)

	proxy.Start()
	rpc.Start()

	// http
	http.Start()

	select {}
}
