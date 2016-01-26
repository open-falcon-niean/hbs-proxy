package http

import (
	"net/http"

	"github.com/open-falcon/hbs-proxy/g"
	"github.com/open-falcon/hbs-proxy/proxy"
)

func configProcHttpRoutes() {
	http.HandleFunc("/proc/config", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, g.Config())
	})

	http.HandleFunc("/proc/counters", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, make([]interface{}, 0))
	})

	http.HandleFunc("/proc/hbs/pools", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, proxy.ConnPools.Proc())
	})
}
