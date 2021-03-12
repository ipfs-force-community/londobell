module github.com/dtynn/londobell

go 1.15

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-paramfetch v0.0.2-0.20200701152213-3e0f0afdc261
	github.com/filecoin-project/go-state-types v0.1.0
	github.com/filecoin-project/lotus v1.5.0
	github.com/filecoin-project/specs-actors/v2 v2.3.4
	github.com/filecoin-project/specs-actors/v3 v3.0.3
	github.com/go-redis/redis/v8 v8.6.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ipfs-blockstore v1.0.3
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/ipfs/go-metrics-prometheus v0.0.2
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multihash v0.0.14
	github.com/prometheus/client_golang v1.6.0
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac
	github.com/urfave/cli/v2 v2.2.0
	go.mongodb.org/mongo-driver v1.5.0
	go.opencensus.io v0.22.5
	go.uber.org/fx v1.9.0
	go.uber.org/zap v1.16.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
