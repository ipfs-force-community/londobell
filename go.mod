module github.com/dtynn/londobell

go 1.16

require (
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/lotus v1.10.1
	github.com/filecoin-project/specs-actors/v2 v2.3.5-0.20210114162132-5b58b773f4fb
	github.com/filecoin-project/specs-actors/v3 v3.1.0
	github.com/filecoin-project/specs-actors/v4 v4.0.0
	github.com/filecoin-project/specs-actors/v5 v5.0.1 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multihash v0.0.14
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.10.5 // indirect
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.2.0
	go.mongodb.org/mongo-driver v1.5.0
	go.uber.org/fx v1.9.0
	go.uber.org/zap v1.16.0
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
