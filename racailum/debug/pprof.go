package debug

import (
	"net/http"
	"net/http/pprof"
	"runtime"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("ra-debug")
var rpclog = log

func Setup(mux *http.ServeMux) {
	log.Info("setup http handlers for debug")

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof-set/block", handleFractionOpt("BlockProfileRate", runtime.SetBlockProfileRate))
	mux.Handle("/debug/pprof-set/mutex", handleFractionOpt("MutexProfileFraction", func(x int) {
		runtime.SetMutexProfileFraction(x)
	}))
}
