package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

var hlog = log.With("http", true)

var (
	ctxType = reflect.TypeOf(new(context.Context)).Elem()
)

func mustHandler(v interface{}) http.HandlerFunc {
	h, err := handler(v)
	if err != nil {
		panic(fmt.Errorf("construct http handler: %w", err))
	}

	return h
}

// Handler attempts to convert a func(req) (resp, error) to http.HandlerFunc
func handler(v interface{}) (http.HandlerFunc, error) {
	rv := reflect.ValueOf(v)
	if rk := rv.Kind(); rk != reflect.Func {
		return nil, fmt.Errorf("expected a func, got %s", rk)
	}

	rt := rv.Type()
	numIn := rt.NumIn()
	if numIn > 2 {
		return nil, fmt.Errorf("expected at most 2 parameter, got %d", numIn)
	}

	numOut := rt.NumOut()
	if numOut > 1 {
		return nil, fmt.Errorf("expected at most 1 output, got %d", numIn)
	}

	var reqType reflect.Type
	hasCtx := false

	switch numIn {
	case 1:
		first := rt.In(0)
		if first == ctxType {
			hasCtx = true
		} else {
			reqType = first
		}

	case 2:
		if first := rt.In(0); first != ctxType {
			return nil, fmt.Errorf("first param should be context.Context, got %s", first)
		}

		hasCtx = true
		reqType = rt.In(1)
	}

	return func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		in := make([]reflect.Value, 0, 2)
		if hasCtx {
			in = append(in, reflect.ValueOf(r.Context()))
		}

		if reqType != nil {
			recv := reflect.New(reqType)
			err := json.NewDecoder(r.Body).Decode(recv.Interface())
			maybeAbort(err)

			in = append(in, recv.Elem())
		}

		out := rv.Call(in)
		if numOut == 1 {
			err := json.NewEncoder(rw).Encode(out[0].Interface())
			maybeAbort(err)
		}

	}, nil
}

func maybeAbort(err error) {
	if err == nil {
		return
	}

	panic(err)
}

func cors(inner http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
		rw.Header().Set("Access-Control-Allow-Methods", "*")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			return
		}

		inner.ServeHTTP(rw, r)
	}
}

func safe(inner http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				hlog.Errorf("internal error: %s", e)
			}
		}()

		inner.ServeHTTP(rw, r)
	}
}
