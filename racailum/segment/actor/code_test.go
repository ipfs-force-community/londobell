package actor

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"

	"github.com/filecoin-project/lotus/build"
)

var testBuilder = cid.V1Builder{Codec: cid.Raw, MhType: mh.IDENTITY}

func TestActorConvert(t *testing.T) {
	actorMap := generateDefaultActorMap()

	builtinMap := map[string]cid.Cid{}
	for code, name := range actorMap {
		builtinMap[name] = code
	}

	onlyNames := []string{"fil/2/v2only", "fil/3/v3only"}
	onlyCodes := []cid.Cid{}
	for _, name := range onlyNames {
		code, err := testBuilder.Sum([]byte(name))
		if err != nil {
			t.Fatalf("generate code for %s: %s", name, err)
		}

		onlyCodes = append(onlyCodes, code)
		actorMap[code] = name
	}

	vnames := []string{
		"fil/1/storageminer",
		"fil/2/storageminer",
		"fil/3/storageminer",
	}

	vcodes := []cid.Cid{}
	for _, vn := range vnames {
		vcodes = append(vcodes, builtinMap[vn])
	}

	cases := []struct {
		epoch        abi.ChainEpoch
		name         string
		ok           bool
		expectedCode cid.Cid
		expectedName string
	}{
		{
			epoch:        0,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        0,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        0,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height - 1,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height - 1,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height - 1,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},

		{
			epoch:        build.UpgradeActorsV2Height,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[0],
			expectedName: vnames[0],
		},
		{
			epoch:        build.UpgradeActorsV2Height + 1,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV2Height + 1,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV3Height - 1,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV3Height - 1,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV3Height - 1,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},

		{
			epoch:        build.UpgradeActorsV3Height,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV3Height,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV3Height,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[1],
			expectedName: vnames[1],
		},
		{
			epoch:        build.UpgradeActorsV3Height + 1,
			name:         vnames[0],
			ok:           true,
			expectedCode: vcodes[2],
			expectedName: vnames[2],
		},
		{
			epoch:        build.UpgradeActorsV3Height + 1,
			name:         vnames[1],
			ok:           true,
			expectedCode: vcodes[2],
			expectedName: vnames[2],
		},
		{
			epoch:        build.UpgradeActorsV3Height + 1,
			name:         vnames[2],
			ok:           true,
			expectedCode: vcodes[2],
			expectedName: vnames[2],
		},
		{
			epoch: 0,
			name:  onlyNames[0],
			ok:    false,
		},
		{
			epoch: build.UpgradeActorsV2Height - 1,
			name:  onlyNames[0],
			ok:    false,
		},
		{
			epoch:        build.UpgradeActorsV2Height,
			name:         onlyNames[0],
			ok:           true,
			expectedCode: onlyCodes[0],
			expectedName: onlyNames[0],
		},
		{
			epoch:        build.UpgradeActorsV3Height - 1,
			name:         onlyNames[0],
			ok:           true,
			expectedCode: onlyCodes[0],
			expectedName: onlyNames[0],
		},
		{
			epoch:        build.UpgradeActorsV3Height,
			name:         onlyNames[0],
			ok:           true,
			expectedCode: onlyCodes[0],
			expectedName: onlyNames[0],
		},

		{
			epoch: 0,
			name:  onlyNames[1],
			ok:    false,
		},
		{
			epoch: build.UpgradeActorsV2Height - 1,
			name:  onlyNames[1],
			ok:    false,
		},
		{
			epoch: build.UpgradeActorsV2Height,
			name:  onlyNames[1],
			ok:    false,
		},
		{
			epoch: build.UpgradeActorsV3Height - 1,
			name:  onlyNames[1],
			ok:    false,
		},
		{
			epoch:        build.UpgradeActorsV3Height,
			name:         onlyNames[1],
			ok:           true,
			expectedCode: onlyCodes[1],
			expectedName: onlyNames[1],
		},
	}

	convertor := generateActorCodeConvertor(defaultActorUpgradeSchedule, actorMap)
	for i, c := range cases {
		code, name, ok := convertor(c.epoch, c.name)
		if ok != c.ok {
			t.Fatalf("#%d expecting converted for %s@%d to be %v, got %v", i, c.name, c.epoch, c.ok, ok)
		}

		if !ok {
			continue
		}

		if code != c.expectedCode {
			t.Fatalf("#%d expecting code %s, got %s", i, c.expectedCode, code)
		}

		if name != c.expectedName {
			t.Fatalf("#%d expecting name %s, got %s", i, c.expectedName, name)
		}
	}
}
