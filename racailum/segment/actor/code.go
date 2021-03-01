package actor

import (
	"sort"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/stmgr"
)

func init() {
	DefaultActorConvertor = generateActorCodeConvertor(defaultActorUpgradeSchedule, generateDefaultActorMap())
}

// (prevUpgradeHeight, currentUpgradeHeight]
var defaultActorUpgradeSchedule = map[string]abi.ChainEpoch{
	"fil/1/": 0,
	"fil/2/": build.UpgradeActorsV2Height,
	"fil/3/": build.UpgradeActorsV3Height,
}

// DefaultActorConvertor is the default actor covertor based on specs-actors' upgrade schedule
var DefaultActorConvertor actorCodeConvertor

type actorVersionInfo struct {
	epoch abi.ChainEpoch
	code  cid.Cid
	full  string
}

type actorCodeConvertor func(abi.ChainEpoch, string) (cid.Cid, string, bool)

func generateDefaultActorMap() map[cid.Cid]string {
	actorMap := map[cid.Cid]string{}
	for code := range stmgr.MethodsMap {
		actorMap[code] = builtin.ActorNameByCode(code)
	}

	return actorMap
}

func generateActorCodeConvertor(sched map[string]abi.ChainEpoch, actorMap map[cid.Cid]string) actorCodeConvertor {
	baseUps := map[string][]actorVersionInfo{}
	nameMap := map[string]string{}

	for code, full := range actorMap {
		for prefix, upEpoch := range sched {
			if !strings.HasPrefix(full, prefix) {
				continue
			}

			base := full[len(prefix):]
			nameMap[full] = base

			baseUps[base] = append(baseUps[base], actorVersionInfo{
				epoch: upEpoch,
				code:  code,
				full:  full,
			})
		}
	}

	for base := range baseUps {
		ups := baseUps[base]
		sort.Slice(ups, func(i, j int) bool {
			return ups[i].epoch < ups[j].epoch
		})

		baseUps[base] = ups
	}

	upgraders := map[string][]actorVersionInfo{}
	for full, base := range nameMap {
		upgraders[full] = baseUps[base]
	}

	return func(at abi.ChainEpoch, name string) (cid.Cid, string, bool) {
		upgrader, ok := upgraders[name]
		if !ok || len(upgrader) == 0 {
			return cid.Undef, "", false
		}

		if at < upgrader[0].epoch {
			return cid.Undef, "", false
		}

		upSize := len(upgrader)

		for i := 1; i < upSize; i++ {
			if at <= upgrader[i].epoch {
				maybe := upgrader[i-1]
				return maybe.code, maybe.full, true
			}
		}

		last := upgrader[upSize-1]
		return last.code, last.full, true
	}
}
