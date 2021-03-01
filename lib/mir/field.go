package mir

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	emptyValue = reflect.Value{}
)

func getSrcFieldValue(srcv reflect.Value, index []int) (reflect.Value, bool) {
	if len(index) == 0 {
		return emptyValue, false
	}

	fv := srcv.Field(index[0])
	fvKind := fv.Kind()
	for _, i := range index[1:] {
		if fvKind == reflect.Ptr {
			if fv.IsNil() {
				return emptyValue, false
			}

			fv = fv.Elem()
			fvKind = fv.Kind()
		}

		if fvKind != reflect.Struct {
			return emptyValue, false
		}

		fv = fv.Field(i)
		fvKind = fv.Kind()
	}

	return fv, true
}

func getFieldMapping(dst, src reflect.Type) fieldMappingRes {
	m, ok := reg.lookupFieldMapping(dst, src)
	if !ok {
		m = mapFields(dst, src)
	}

	reg.registerFieldMapping(dst, src, m)
	return m
}

type fieldMappingRes struct {
	mapping []fieldMapping
	errs    []error
}

type fieldMapping struct {
	rawname  string
	index    int
	srcIndex []int
}

func mapFields(dst reflect.Type, src reflect.Type) fieldMappingRes {
	numFields := dst.NumField()

	res := fieldMappingRes{
		mapping: make([]fieldMapping, 0, numFields),
	}

	for i := 0; i < numFields; i++ {
		field := dst.Field(i)
		name := field.Name
		required := false

		tag, tok := field.Tag.Lookup("mir")
		if tok {
			pieces := strings.Split(tag, ",")
			if s := strings.TrimSpace(pieces[0]); s != "" {
				// ignored field
				if s == "-" {
					continue
				}

				name = s
			}

			for _, p := range pieces[1:] {
				switch strings.TrimSpace(p) {
				case "required":
					required = true
				}
			}
		}

		if !isPublic(name) {
			continue
		}

		srcField, ok := src.FieldByName(name)
		if !ok {
			if required {
				res.errs = append(res.errs, fmt.Errorf("src field %s is required for dest %s", name, field.Name))
			}

			continue
		}

		res.mapping = append(res.mapping, fieldMapping{
			rawname:  field.Name,
			index:    i,
			srcIndex: srcField.Index,
		})
	}

	return res
}

func isPublic(name string) bool {
	if name == "" {
		return false
	}

	b := name[0]
	return b >= 'A' && b <= 'Z'
}

func copyAppendIndexes(prev []int, idx ...int) []int {
	next := make([]int, len(prev)+len(idx))
	copy(next[:len(prev)], prev)
	copy(next[len(prev):], idx)
	return next
}
