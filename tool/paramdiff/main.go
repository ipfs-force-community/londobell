package main

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
)

const pkgPrefix = "github.com/filecoin-project/specs-actors/"

func main() {
	latest := actor.Specs[len(actor.Specs)-1]
	latestVer := actors.Versions[len(actor.Specs)-1]

	latestPrefix := fmt.Sprintf("%sv%d/", pkgPrefix, latestVer)

	types := map[string]reflect.Type{}
	for _, act := range latest {
		methods := act.Exports()
		for _, meth := range methods {
			methType := reflect.TypeOf(meth)
			if methType == nil || methType.Kind() != reflect.Func {
				continue
			}

			for ii := 0; ii < methType.NumIn(); ii++ {
				extractFields(types, methType.In(ii))
			}

			for oi := 0; oi < methType.NumOut(); oi++ {
				extractFields(types, methType.In(oi))
			}
		}
	}

	typeSlice := make([]reflect.Type, 0, len(types))
	for _, typ := range types {
		typeSlice = append(typeSlice, typ)
	}

	sort.Slice(typeSlice, func(i, j int) bool {
		pkgi := typeSlice[i].PkgPath()
		pkgj := typeSlice[j].PkgPath()

		if pkgi != pkgj {
			return pkgi < pkgj
		}

		return typeSlice[i].String() < typeSlice[j].String()
	})

	for _, typ := range typeSlice {
		pkgPath := typ.PkgPath()
		if strings.HasPrefix(pkgPath, latestPrefix) {
			pkgPath = strings.ReplaceAll(pkgPath, latestPrefix, "github.com/filecoin-project/specs-actors/latest/")
		}

		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", typ.Kind(), typ.String(), pkgPath)
	}
}

func extractFields(types map[string]reflect.Type, typ reflect.Type) {
	kind := typ.Kind()

INNER:
	for {
		switch kind {
		case reflect.Ptr, reflect.Array, reflect.Slice:
			typ = typ.Elem()
			kind = typ.Kind()

		default:
			break INNER
		}
	}

	pkgPath := typ.PkgPath()
	typStr := typ.String()

	// primitive types
	if pkgPath == "" && !strings.Contains(typStr, ".") {
		return
	}

	ident := fmt.Sprintf("%s-%s", pkgPath, typStr)
	if _, has := types[ident]; has {
		return
	}

	types[ident] = typ
	if pkgPath != "" && !strings.HasPrefix(pkgPath, pkgPrefix) {
		return
	}

	if kind == reflect.Struct {
		for si := 0; si < typ.NumField(); si++ {
			extractFields(types, typ.Field(si).Type)
		}
	}
}
