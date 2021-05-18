package utils

import (
	"net/http"
	_ "net/http/pprof"
)

func HTTPServerPProf() {
	http.ListenAndServe("0.0.0.0:6060", nil)
}
