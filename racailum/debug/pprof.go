package debug

import (
	"net"
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
	mux.Handle("/debug/pprof-set/block", localOnly(handleFractionOpt("BlockProfileRate", runtime.SetBlockProfileRate)))
	mux.Handle("/debug/pprof-set/mutex", localOnly(handleFractionOpt("MutexProfileFraction", func(x int) {
		runtime.SetMutexProfileFraction(x)
	})))
}

// localOnly 限制仅允许本地回环地址访问，防止远程修改运行时 profiling 参数
func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			http.Error(rw, "forbidden: only localhost allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(rw, r)
	})
}
