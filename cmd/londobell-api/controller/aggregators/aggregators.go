package aggregators

import monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

var (
	addressAggregator          []byte
	aggPreNetfeeAggregator     []byte
	aggProNetfeeAggregator     []byte
	blockAggregator            []byte
	finalHeightAggregator      []byte
	minerBlockrewardAggregator []byte
	minersInfoAggregator       []byte
	minersMinedAggregator      []byte
	multisigMessageAggregator  []byte
	punishmentAggregator       []byte
	wincountZlAggregator       []byte
	tracesAggregator           []byte
)

func InitAggregators() {
	// todo: 定期重新读取，无感知变化 or 每次变化重启
	addressAggregator = monitor.GetAddressAggregator()
	aggPreNetfeeAggregator = monitor.GetAggPreNetfeeAggregator()
	aggProNetfeeAggregator = monitor.GetAggProNetfeeAggregator()
	blockAggregator = monitor.GetBlockAggregator()
	finalHeightAggregator = monitor.GetFinalHeightAggregator()
	minerBlockrewardAggregator = monitor.GetMinerBlockrewardAggregator()
	minersInfoAggregator = monitor.GetMinersInfoAggregator()
	minersMinedAggregator = monitor.GetMinersMinedAggregator()
	multisigMessageAggregator = monitor.GetMultisigMessageAggregator()
	punishmentAggregator = monitor.GetPunishmentAggregator()
	wincountZlAggregator = monitor.GetWincountZlAggregator()
	tracesAggregator = monitor.GetTracesAggregator()
}
