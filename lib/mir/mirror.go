package mir

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
)

// common errors
var (
	ErrInvalidDst         = fmt.Errorf("invalid dst")
	ErrInvalidSrc         = fmt.Errorf("invalid src")
	ErrInvalidField       = fmt.Errorf("invalid field")
	ErrValueNotChangeable = fmt.Errorf("value not changeable")
	ErrMismatchedKind     = fmt.Errorf("mismatched kind")
	ErrMismatchedType     = fmt.Errorf("mismatched type")
)

// Mirror trie to apply src's values on to dst
// it tries to handle situations like:
//    - struct T => struct T'
//    - []T => []T'
//    - map[KT]T => map[KT]T'
func Mirror(dst, src interface{}) error {
	dstv := reflect.ValueOf(dst)
	srcv := reflect.ValueOf(src)
	dstKind := dstv.Kind()
	if !valueKindValid(dstKind) {
		return fmt.Errorf("%w: %s", ErrInvalidDst, dstKind)
	}

	srcKind := srcv.Kind()
	if !valueKindValid(srcKind) {
		return fmt.Errorf("%w: %s", ErrInvalidSrc, srcKind)
	}

	return makeMirrorFunc(dstv.Type(), srcv.Type())("destination", dstv, srcv)
}

func makeMirrorFunc(dst, src reflect.Type) (mirfn mirrorFunc) {
	if fn, ok := reg.lookup(dst, src); ok {
		return fn
	}

	defer func() {
		if mirfn != nil {
			reg.register(dst, src, mirfn)
		}
	}()

	if dst == src {
		return func(name string, dstv, srcv reflect.Value) error {
			return safeSet(name, dstv, srcv)
		}
	}

	if src.ConvertibleTo(dst) {
		return func(name string, dstv, srcv reflect.Value) error {
			return safeSet(name, dstv, srcv.Convert(dst))
		}
	}

	skind, dkind := src.Kind(), dst.Kind()
	if !valueKindValid(dkind) {
		return makeMirrorFuncForError(fmt.Errorf("%w: kind %s", ErrInvalidDst, dkind))
	}

	if !valueKindValid(skind) {
		return makeMirrorFuncForError(fmt.Errorf("%w: kind %s", ErrInvalidSrc, skind))
	}

	if dkind == reflect.Interface {
		if !src.Implements(dst) {
			return makeMirrorFuncForError(fmt.Errorf("%w: src type %s does not implement dst type %s", ErrInvalidSrc, src, dst))
		}

		return func(name string, dstv, srcv reflect.Value) error {
			return safeSet(name, dstv, srcv)
		}
	}

	// for src, we should look into it's inner type
	if skind == reflect.Interface || skind == reflect.Ptr {
		return func(name string, dstv, srcv reflect.Value) error {
			if srcv.IsNil() {
				return nil
			}

			srcElem := srcv.Elem()
			return makeMirrorFunc(dst, srcElem.Type())(name, dstv, srcElem)
		}
	}

	if dkind == reflect.Ptr {
		return func(name string, dstv, srcv reflect.Value) error {
			if dstv.IsNil() {
				newv := reflect.New(dst.Elem())
				if err := safeSet(name, dstv, newv); err != nil {
					return err
				}
			}

			return makeMirrorFunc(dst.Elem(), src)(name, dstv.Elem(), srcv)
		}
	}

	if dkind != skind {
		return makeMirrorFuncForError(fmt.Errorf("%w: dst %s, src %s", ErrMismatchedKind, dkind, skind))
	}

	switch dkind {
	case reflect.Array:
		return makeMirrorFuncForArray(dst, src)

	case reflect.Slice:
		return makeMirrorFuncForSlice(dst, src)

	case reflect.Map:
		return makeMirrorFuncForMap(dst, src)

	case reflect.Struct:
		return makeMirrorFuncForStruct(dst, src)

	default:
		panic(fmt.Errorf("unreachable kind %s", dkind))
	}
}

func makeMirrorFuncForError(err error) mirrorFunc {
	return func(_ string, _, _ reflect.Value) error {
		return err
	}
}

func makeMirrorFuncForArray(dst reflect.Type, src reflect.Type) mirrorFunc {
	dstLen, srcLen := dst.Len(), src.Len()
	if dstLen != srcLen {
		return makeMirrorFuncForError(fmt.Errorf("%w: array of different length, %d != %d", ErrMismatchedType, dstLen, srcLen))
	}

	eleMirrorFunc := makeMirrorFunc(dst.Elem(), src.Elem())

	return func(name string, dstv, srcv reflect.Value) error {
		for i := 0; i < dstLen; i++ {
			subname := fmt.Sprintf("{#%d in array (%s)(%s)}", i, dst, name)
			if err := eleMirrorFunc(subname, dstv.Index(i), srcv.Index(i)); err != nil {
				return fmt.Errorf("#%d element %s array: %w", i, dst, err)
			}
		}

		return nil
	}
}

func makeMirrorFuncForSlice(dst reflect.Type, src reflect.Type) mirrorFunc {
	dstElem := dst.Elem()
	eleMirrorFunc := makeMirrorFunc(dstElem, src.Elem())
	return func(name string, dstv, srcv reflect.Value) error {
		srcLen := srcv.Len()

		newSlice := reflect.MakeSlice(dst, srcLen, srcLen)
		if err := safeSet(name, dstv, newSlice); err != nil {
			return fmt.Errorf("init dst slice: %w", err)
		}

		for i := 0; i < srcLen; i++ {
			subname := fmt.Sprintf("{#%d in slice (%s)(%s)}", i, dst, name)
			if err := eleMirrorFunc(subname, dstv.Index(i), srcv.Index(i)); err != nil {
				return fmt.Errorf("#%d(%d) element in slice %s: %w", i, srcLen, dst, err)
			}
		}

		return nil
	}
}

func makeMirrorFuncForMap(dst reflect.Type, src reflect.Type) mirrorFunc {
	dstKeyType := dst.Key()
	srcKeyType := src.Key()
	if dstKeyType != srcKeyType {
		return makeMirrorFuncForError(fmt.Errorf("%w: map of different key type, %s != %s", ErrMismatchedType, dstKeyType, srcKeyType))
	}

	dstElemType := dst.Elem()
	eleMirrorFunc := makeMirrorFunc(dstElemType, src.Elem())

	return func(name string, dstv, srcv reflect.Value) error {
		newMap := reflect.MakeMap(dst)
		if err := safeSet(name, dstv, newMap); err != nil {
			return fmt.Errorf("init dst map: %w", err)
		}

		iter := srcv.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()

			subname := fmt.Sprintf("{key %s in (%s)(%s)}", key, dst, name)
			newElem := reflect.New(dstElemType)
			if err := eleMirrorFunc(subname, newElem.Elem(), val); err != nil {
				return fmt.Errorf("entry %s in map %s: %w", key, dst, err)
			}

			dstv.SetMapIndex(key, newElem.Elem())
		}

		return nil
	}
}

func makeMirrorFuncForStruct(dst reflect.Type, src reflect.Type) mirrorFunc {
	mres := getFieldMapping(dst, src)
	if len(mres.errs) > 0 {
		return makeMirrorFuncForError(fmt.Errorf("%w: %s", ErrInvalidField, multierror.ListFormatFunc(mres.errs)))
	}

	return func(name string, dstv, srcv reflect.Value) error {
		for _, m := range mres.mapping {
			fsrc, ok := getSrcFieldValue(srcv, m.srcIndex)
			if !ok {
				continue
			}

			fdst := dstv.Field(m.index)

			mirfn := makeMirrorFunc(fdst.Type(), fsrc.Type())

			subname := fmt.Sprintf("{field %s in (%s)(%s)}", m.rawname, dst, name)
			if err := mirfn(subname, fdst, fsrc); err != nil {
				return err
			}
		}

		return nil
	}
}

func safeSet(name string, dst, src reflect.Value) error {
	if !dst.CanSet() {
		return fmt.Errorf("%w: for %s of type %s", ErrValueNotChangeable, name, dst.Type())
	}

	dst.Set(src)
	return nil
}

func valueKindValid(vk reflect.Kind) bool {
	valid := true

	switch vk {
	// basic types
	case reflect.Bool,
		reflect.Complex64, reflect.Complex128,
		reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.String,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:

	// array
	case reflect.Array:

	// slice
	case reflect.Slice:

	// for structs, we should walk through all the fields
	case reflect.Struct:

	case reflect.Map:

	case reflect.Interface:

	case reflect.Ptr:

	// wont handle these types
	case reflect.Chan,
		reflect.Func,
		reflect.Invalid,
		reflect.UnsafePointer:

		valid = false

	default:
		panic(fmt.Errorf("unreachable kind %s", vk))
	}

	return valid
}
