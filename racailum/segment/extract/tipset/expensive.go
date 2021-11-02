package tipset

// var expensiveNetworkVersions = map[network.Version]struct{}{}

// func isExpensive(ctx context.Context, stm common.StateManager, ts *common.LinkedTipSet) bool {
//     prev := stm.GetNtwkVersion(ctx, ts.Parent.Height())
//     next := stm.GetNtwkVersion(ctx, ts.Height())
//     if prev == next {
//         return false
//     }

//     _, is := expensiveNetworkVersions[next]
//     return is
// }
