package internal

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
)

func ResolveSpecs(expectedVer int, exported []builtin.RegistryEntry) (*Specs, error) {
	actors := map[string]resolvedActor{}

	for _, actor := range exported {
		if actor.State() != nil {
			stateTyp := reflect.TypeOf(actor.State()).Elem()
			ver, name, err := getStateVer(stateTyp.PkgPath())
			if err != nil {
				return nil, fmt.Errorf("get specs version for %s in %s: %w", stateTyp, stateTyp.PkgPath(), err)
			}

			if ver != expectedVer {
				return nil, fmt.Errorf("version not match, got %d for %s in %s", ver, stateTyp, stateTyp.PkgPath())
			}

			actors[name] = resolvedActor{
				RegistryEntry: actor,
				rtype:         stateTyp,
			}
		}
	}

	return &Specs{
		Ver:      expectedVer,
		Exported: exported,
		actors:   actors,
	}, nil
}

type SpecsDiff struct {
	Adds         []reflect.Type
	Minuses      []reflect.Type
	StateChanges []StateDiff
}

type StateDiff struct {
	PrevActor reflect.Type
	NextActor reflect.Type
	StructDiff
}

type resolvedActor struct {
	builtin.RegistryEntry
	rtype reflect.Type
}

type Specs struct {
	Ver      int
	Exported []builtin.RegistryEntry
	actors   map[string]resolvedActor
}

func CompareSpecs(prev, next *Specs) SpecsDiff {
	var sdiff SpecsDiff
	for name, pactor := range prev.actors {
		var nactor resolvedActor
		var has bool
		nactor, has = next.actors[name]
		if !has {
			sdiff.Minuses = append(sdiff.Minuses, pactor.rtype)
			continue
		}

		// check states
		pstateTyp := reflect.TypeOf(pactor.State()).Elem()
		nstateTyp := reflect.TypeOf(nactor.State()).Elem()

		fields := CompareStructs(pstateTyp, nstateTyp)
		if fields.IsEmpty() {
			continue
		}

		sdiff.StateChanges = append(sdiff.StateChanges, StateDiff{
			PrevActor:  pactor.rtype,
			NextActor:  nactor.rtype,
			StructDiff: fields,
		})
	}

	for name, nactor := range next.actors {
		if _, has := prev.actors[name]; !has {
			sdiff.Adds = append(sdiff.Adds, nactor.rtype)
		}
	}

	sort.Slice(sdiff.Adds, func(i, j int) bool {
		return sdiff.Adds[i].String() < sdiff.Adds[j].String()
	})

	sort.Slice(sdiff.Minuses, func(i, j int) bool {
		return sdiff.Minuses[i].String() < sdiff.Minuses[j].String()
	})

	sort.Slice(sdiff.StateChanges, func(i, j int) bool {
		return sdiff.StateChanges[i].PrevActor.String() < sdiff.StateChanges[j].PrevActor.String()
	})

	return sdiff
}

func getStateVer(pkg string) (int, string, error) {
	splits := strings.Split(pkg, "/")
	name := splits[len(splits)-1]

	if strings.HasPrefix(pkg, "github.com/filecoin-project/specs-actors/") {
		if strings.HasPrefix(pkg, "github.com/filecoin-project/specs-actors/actors") {
			return 0, name, nil
		}

		var ver int
		_, err := fmt.Fscanf(bytes.NewReader([]byte(pkg)), "github.com/filecoin-project/specs-actors/v%d/actors", &ver)
		return ver, name, err
	}

	var ver int
	_, err := fmt.Fscanf(bytes.NewReader([]byte(pkg)), "github.com/filecoin-project/go-state-types/builtin/v%d/"+name, &ver)
	return ver, name, err
}
