package aggregators

import pool_monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

// todo: 确认路径
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
)

func InitAggregators() {
	// todo: 定期重新读取，无感知变化 or 每次变化重启
	addressAggregator = pool_monitor.GetAddressAggregator()
	aggPreNetfeeAggregator = pool_monitor.GetAggPreNetfeeAggregator()
	aggProNetfeeAggregator = pool_monitor.GetAggProNetfeeAggregator()
	blockAggregator = pool_monitor.GetBlockAggregator()
	finalHeightAggregator = pool_monitor.GetFinalHeightAggregator()
	minerBlockrewardAggregator = pool_monitor.GetMinerBlockrewardAggregator()
	minersInfoAggregator = pool_monitor.GetMinersInfoAggregator()
	minersMinedAggregator = pool_monitor.GetMinersMinedAggregator()
	multisigMessageAggregator = pool_monitor.GetMultisigMessageAggregator()
	punishmentAggregator = pool_monitor.GetPunishmentAggregator()
	wincountZlAggregator = pool_monitor.GetWincountZlAggregator()
}
