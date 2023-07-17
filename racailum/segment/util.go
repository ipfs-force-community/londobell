package segment

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

// ExtractLinkedTipSets extracts linked tipsets within range [lower, from); range [lower, from] if tmp is true
func ExtractLinkedTipSets(cs common.ChainStore, from *types.TipSet, lower *abi.ChainEpoch, tmp bool) ([]*common.LinkedTipSet, error) {
	var destEpoch abi.ChainEpoch

	if lower != nil {
		destEpoch = *lower
	}

	h := from.Height()
	if h <= destEpoch {
		return nil, nil
	}

	length := int(h - destEpoch)
	if tmp {
		length = int(h-destEpoch) + 1
	}
	tss := make([]*common.LinkedTipSet, 0, length)
	var prev *types.TipSet
	_, err := TraverseTipSets(cs, from, func(walked *types.TipSet, walkedEpoch abi.ChainEpoch) (bool, error) {
		if walkedEpoch < destEpoch {
			return false, nil
		}

		if tmp {
			// allow nil child
			tss = append(tss, &common.LinkedTipSet{
				TipSet: walked,
				Child:  prev,
			})
		} else {
			if prev != nil {
				tss = append(tss, &common.LinkedTipSet{
					TipSet: walked,
					Child:  prev,
				})
			}
		}

		prev = walked

		return true, nil
	})

	if err != nil {
		return nil, err
	}

	got := len(tss)
	for i := 0; i < got/2; i++ {
		j := got - i - 1
		tss[i], tss[j] = tss[j], tss[i]
	}

	return tss, nil
}

func TraverseTipSets(cs common.ChainStore, curts *types.TipSet, traverseFn func(*types.TipSet, abi.ChainEpoch) (bool, error)) (int, error) {
	count := 0

	for {
		curh := curts.Height()
		keep, err := traverseFn(curts, curh)
		count++
		if err != nil {
			return count, err
		}

		if !keep || curh == 0 {
			return count, nil
		}

		parentTSK := curts.Parents()
		parentTS, err := cs.LoadTipSet(context.Background(), parentTSK)
		if err != nil {
			return count, err
		}

		curts = parentTS
	}
}

func ExtractPrimaryKeyValue(doc interface{}) (interface{}, error) {
	dt := reflect.TypeOf(doc)
	vt := reflect.ValueOf(doc)
	if dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
		vt = vt.Elem()
	}

	if !vt.IsValid() {
		return nil, fmt.Errorf("invalid doc: %+v", doc)
	}

	for i := 0; i < dt.NumField(); i++ {
		if tagstr := dt.Field(i).Tag.Get("bson"); tagstr != "" {
			if name := strings.Split(tagstr, ",")[0]; name == "_id" {
				return vt.Field(i).Interface(), nil
			}
		}
	}

	return nil, fmt.Errorf("field _id not found for %+v", doc)
}

func ExtractEpochField(doc interface{}) (string, error) {
	dt := reflect.TypeOf(doc)
	if dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
	}

	field, ok := dt.FieldByName("Epoch")
	if !ok {
		return "", fmt.Errorf("field Epoch not found for %+v", doc)
	}

	if tagstr := field.Tag.Get("bson"); tagstr != "" {
		if name := strings.Split(tagstr, ",")[0]; name != "" {
			return name, nil
		}
	}

	return field.Name, nil
}

func ExtractEpochValue(doc interface{}) (int64, error) {
	dt := reflect.TypeOf(doc)
	vt := reflect.ValueOf(doc)
	if dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
		vt = vt.Elem()
	}

	if !vt.IsValid() {
		return 0, fmt.Errorf("invalid doc: %+v", doc)
	}

	if _, ok := dt.FieldByName("Epoch"); !ok {
		return 0, fmt.Errorf("field Epoch not found for %+v", doc)
	}

	if vt.FieldByName("Epoch").Type().ConvertibleTo(reflect.TypeOf(int64(0))) {
		return vt.FieldByName("Epoch").Convert(reflect.TypeOf(int64(0))).Int(), nil
	}

	return 0, fmt.Errorf("field Epoch is not convertible to int64: %+v", doc)
}

func ExtractUpdateDoc(doc interface{}) (bson.M, error) {
	dt := reflect.TypeOf(doc)
	vt := reflect.ValueOf(doc)
	if dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
		vt = vt.Elem()
	}

	if !vt.IsValid() {
		return nil, fmt.Errorf("null doc for ExtractUpdateDoc: %+v", doc)
	}

	res := bson.M{}
	for i := 0; i < dt.NumField(); i++ {
		if tagstr := dt.Field(i).Tag.Get("bson"); tagstr != "" {
			if name := strings.Split(tagstr, ",")[0]; name != "" {
				res[name] = vt.Field(i).Interface()
				continue
			}
		}

		res[dt.Field(i).Name] = vt.Field(i).Interface()
	}

	return res, nil
}
