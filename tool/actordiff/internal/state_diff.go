package internal

import (
	"reflect"
	"sort"
	"strings"
)

type StructDiff struct {
	Adds    []reflect.StructField
	Minuses []reflect.StructField
	Changes [][2]reflect.StructField
}

func (sd *StructDiff) IsEmpty() bool {
	return len(sd.Adds) == 0 && len(sd.Minuses) == 0 && len(sd.Minuses) == 0
}

// compareStructs compares two structs and returns their field diffs
// we just let it panic if either prev or next is not a struct
func compareStructs(prev, next reflect.Type) StructDiff {
	prevFields := map[string]reflect.StructField{}
	nextFields := map[string]reflect.StructField{}

	for fi := 0; fi < prev.NumField(); fi++ {
		field := prev.Field(fi)
		prevFields[field.Name] = field
	}

	for fi := 0; fi < next.NumField(); fi++ {
		field := next.Field(fi)

		// not changed
		if pf, has := prevFields[field.Name]; has && pf.Type == field.Type {
			delete(prevFields, field.Name)
			continue
		}

		nextFields[field.Name] = field
	}

	var sdiff StructDiff
PREV_FIELDS:
	for name, pf := range prevFields {
		nf, has := nextFields[name]
		if !has {
			sdiff.Minuses = append(sdiff.Minuses, pf)
			continue
		}

		delete(nextFields, name)

		// detect if they are actually different
		if pf.Type.Kind() == nf.Type.Kind() &&
			strings.HasPrefix(pf.Type.PkgPath(), "github.com/filecoin-project/specs-actors/") &&
			strings.HasPrefix(nf.Type.PkgPath(), "github.com/filecoin-project/specs-actors/") {

			pInside := pf.Type
			pInsideKind := pInside.Kind()
			nInside := nf.Type
			nInsideKind := nInside.Kind()

			for pInsideKind == nInsideKind {
				// TODO: there could be more kinds here
				// for now, we just make ptr & [] special
				if pInsideKind != reflect.Ptr && pInsideKind != reflect.Slice {
					break
				}

				pInside = pInside.Elem()
				pInsideKind = pInside.Kind()

				nInside = nInside.Elem()
				nInsideKind = nInside.Kind()
			}

			if pInsideKind == nInsideKind {
				if pInside.ConvertibleTo(nInside) {
					continue PREV_FIELDS
				}

				if pInsideKind == reflect.Struct {
					insideDiff := compareStructs(pInside, nInside)
					if insideDiff.IsEmpty() {
						continue PREV_FIELDS
					}
				}
			}

		}

		sdiff.Changes = append(sdiff.Changes, [2]reflect.StructField{pf, nf})
	}

	for _, nf := range nextFields {
		sdiff.Adds = append(sdiff.Adds, nf)
	}

	sort.Slice(sdiff.Adds, func(i, j int) bool {
		return sdiff.Adds[i].Name < sdiff.Adds[j].Name
	})

	sort.Slice(sdiff.Minuses, func(i, j int) bool {
		return sdiff.Minuses[i].Name < sdiff.Minuses[j].Name
	})

	sort.Slice(sdiff.Changes, func(i, j int) bool {
		return sdiff.Changes[i][0].Name < sdiff.Changes[j][0].Name
	})

	return sdiff
}
