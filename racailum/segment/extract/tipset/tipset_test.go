package tipset

import (
	"testing"
)

func TestExtract(t *testing.T) {
	//err := extractnew()
	//require.NoError(t, err)

}

//func extractnew() error {
//	EnableExtractMinerSector := true
//	EnableExtractSectorClaim := true
//
//	ctx := context.TODO()
//
//	//fullnode.API = fullnode.NewAppropriateAPI([]util.Node{util.Node{URL: "ws://106.14.10.70:1234/rpc/v0"}})
//	//err := fullnode.API.Choose(ctx)
//	//if err != nil {
//	//	return err
//	//}
//	//api := fullnode.API.GetAppropriateAPI()
//
//	api, _, err := client.NewFullNodeRPCV0(ctx, "ws://106.14.10.70:1234/rpc/v0", nil)
//	if err != nil {
//		return err
//	}
//
//	// 3045420
//	c, err := cid.Decode("bafy2bzacedhulkfmqnlik4z4ac3ysldtk22oanagkiicsrc26tcwzhakwtncg")
//	if err != nil {
//		return err
//	}
//	c2, err := cid.Decode("bafy2bzacebbwo3jfk6qwxx6fumq4xoofysf56bzdbborytvuioj3m2oy4gzl4")
//	if err != nil {
//		return err
//	}
//
//	// child 3045421
//	cc, err := cid.Decode("bafy2bzacecdxxod2s5lwcrx2rxbv2mmeodqqg27rp2rqmjzowzzmd7bsae4ly")
//	if err != nil {
//		return err
//	}
//	cc2, err := cid.Decode("bafy2bzacecn7nyl7aoqbk6qnfgml55qzcz7pzlfwuccjax4xilipm4edim2jg")
//	if err != nil {
//		return err
//	}
//	cc3, err := cid.Decode("bafy2bzacedzy2gxe6pfi6bbs236eyxwl2w2erzolawmbzpsevh3xu6si22ptq")
//	if err != nil {
//		return err
//	}
//	cc4, err := cid.Decode("bafy2bzacebu6tua2llls6se4soy3cp56n7v4wa7oi3443d24dwltufurtelay")
//	if err != nil {
//		return err
//	}
//	cc5, err := cid.Decode("bafy2bzaceb47gmuqlprlwwtw4sonvqgx4go6dsk3keqpzz347myor6iv6k5mo")
//	if err != nil {
//		return err
//	}
//	cc6, err := cid.Decode("bafy2bzaceb4edytqpatv6zpxcqkdbenewtvhity5hycuww5svuiknkiaqrzam")
//	if err != nil {
//		return err
//	}
//	cc7, err := cid.Decode("bafy2bzaceb6pchuxdv7x2oa6ustxiktu54mmpjx4wxtavxkxsui57agddrkh6")
//	if err != nil {
//		return err
//	}
//	cc8, err := cid.Decode("bafy2bzacedcjnmtotp4oogs53wuynv5nfkguzslkjsuuux3ndkgig7xuvf4lu")
//	if err != nil {
//		return err
//	}
//
//	cids := []cid.Cid{c, c2}
//	ccids := []cid.Cid{cc, cc2, cc3, cc4, cc5, cc6, cc7, cc8}
//
//	ts, err := api.ChainGetTipSet(ctx, types.NewTipSetKey(cids...))
//	if err != nil {
//		return err
//	}
//	cts, err := api.ChainGetTipSet(ctx, types.NewTipSetKey(ccids...))
//	if err != nil {
//		return err
//	}
//
//	cso, err := api.StateCompute(ctx, 3045420, nil, types.NewTipSetKey(cids...))
//	if err != nil {
//		return err
//	}
//
//	res := make([]common.Document, 0, 4096)
//
//	var invocs []common.InvocResultCompact
//	if err := mir.Mirror(&invocs, cso.Trace); err != nil {
//		panic(fmt.Errorf("mirroring exec invoc results: %w", err))
//	}
//
//	etraces := make([]persistExecTrace, 0, len(invocs)*4)
//	gasTraceNames := map[string]struct{}{}
//
//	store := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))
//
//	for i := range invocs {
//		exec := &invocs[i].ExecutionTrace
//		etraces = append(etraces, persistExecTrace{
//			seq:    []int{i},
//			parent: nil,
//			exec:   exec,
//			gas:    &invocs[i].GasCost,
//		})
//
//		for ni := range exec.GasCharges {
//			name := exec.GasCharges[ni].Name
//			if _, has := gasTraceNames[name]; !has {
//				gasTraceNames[name] = struct{}{}
//			}
//		}
//
//		walkExecTrace([]int{i}, exec, func(subseq []int, subparent, subexec *common.ExecutionTraceCompact) {
//			etraces = append(etraces, persistExecTrace{
//				seq:    copyIndexes(subseq),
//				parent: subparent,
//				exec:   subexec,
//			})
//
//			for ni := range subexec.GasCharges {
//				name := subexec.GasCharges[ni].Name
//				if _, has := gasTraceNames[name]; !has {
//					gasTraceNames[name] = struct{}{}
//				}
//			}
//		})
//	}
//
//	for i := range etraces {
//		p := etraces[i]
//		msg := &p.exec.Msg
//
//		var actorName string
//		mact, err := api.StateGetActor(ctx, msg.To, ts.Key())
//		if err != nil {
//			return err
//		}
//
//		if mact.Code == build.MustParseCid("bafk2bzacec24okjqrp7c7rj3hbrs5ez5apvwah2ruka6haesgfngf37mhk6us") {
//			actorName = "fil/11/storageminer"
//		} else if mact.Code == build.MustParseCid("bafk2bzacedej3dnr62g2je2abmyjg3xqv4otvh6e26du5fcrhvw7zgcaaez3a") {
//			actorName = "fil/11/verifiedregistry"
//		}
//
//		mi, err := util.LookupMethodInfo(ts.Height(), msg.Method, msg.From.String()[1:], msg.To.String()[1:], actorName)
//		if err != nil {
//			if err.Error() == "actor method not found" {
//				panic(fmt.Errorf("lookup method info for %s/%d: %w", msg.To, msg.Method, err))
//			}
//		}
//		var (
//			minerID address.Address
//			cmas    lminer.State
//		)
//
//		if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") {
//			minerID, err = api.StateLookupID(ctx, msg.To, ts.Key())
//			if err != nil {
//				return err
//			}
//
//			cmact, err := api.StateGetActor(ctx, msg.To, cts.Key())
//			if err != nil {
//				return err
//			}
//
//			cmas, err = lminer.Load(store, cmact)
//			if err != nil {
//				return err
//			}
//		}
//
//		vact, err := api.StateGetActor(ctx, builtin.VerifiedRegistryActorAddr, cts.Key())
//		if err != nil {
//			return err
//		}
//
//		cvas, err := verifreg.Load(store, vact)
//		if err != nil {
//			return err
//		}
//
//		mkact, err := api.StateGetActor(ctx, builtin.StorageMarketActorAddr, cts.Key())
//		if err != nil {
//			return err
//		}
//
//		cmkas, err := market.Load(store, mkact)
//		if err != nil {
//			return err
//		}
//
//		// ProveCommitSector & ProveCommitAggregate generate部分sector
//		if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && (p.exec.Msg.Method == builtintypes.MethodsMiner.ProveCommitSector || p.exec.Msg.Method == builtintypes.MethodsMiner.ProveCommitAggregate) {
//			var sectorNumbers = bitfield.New()
//			// 解析参数，version
//			switch p.exec.Msg.Method {
//			case builtintypes.MethodsMiner.ProveCommitSector:
//				var params miner.ProveCommitSectorParams // todo
//				if err := params.UnmarshalCBOR(bytes.NewReader(p.exec.Msg.Params)); err != nil {
//					return fmt.Errorf("unmarshal ProveCommitSectorParams for %v failed: %v", p.exec.Msg.Cid(), err)
//				}
//
//				sectorNumbers.Set(uint64(params.SectorNumber))
//			case builtintypes.MethodsMiner.ProveCommitAggregate:
//				var params miner.ProveCommitAggregateParams
//				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//					return fmt.Errorf("unmarshal ProveCommitAggregateParams for %v failed: %v", msg.Cid(), err)
//				}
//
//				sectorNumbers = params.SectorNumbers
//			default:
//				return fmt.Errorf("invalid method: %v", p.exec.Msg.Method)
//			}
//
//			if EnableExtractMinerSector {
//				fmt.Println(msg.Cid())
//				// get sectors for sectorNumbers from state
//				var sectorInfos []*lminer.SectorOnChainInfo
//				if err := sectorNumbers.ForEach(func(i uint64) error {
//					sectorInfo, err := cmas.GetSector(abi.SectorNumber(i))
//					if err != nil {
//						// err is nil when not found
//						return fmt.Errorf("get sector %v for %v failed: %v", i, msg.To, err)
//					}
//
//					if sectorInfo != nil {
//						sectorInfos = append(sectorInfos, sectorInfo)
//					}
//
//					return nil
//				}); err != nil {
//					return fmt.Errorf("load sector info for %v at %v failed: %v", msg.To, ts.Height(), err)
//				}
//
//				for _, info := range sectorInfos {
//					mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, false, ts.Height())
//					res = append(res, mst)
//				}
//			}
//
//			if EnableExtractSectorClaim {
//				fmt.Println(msg.Cid())
//				// get claims for sectors
//				claims, err := cvas.GetClaims(minerID)
//				if err != nil {
//					return fmt.Errorf("failed to get claims for provider: %v: %v", minerID, err)
//				}
//
//				if err := sectorNumbers.ForEach(func(i uint64) error {
//					for claimID, claim := range claims {
//						if claim.Sector == abi.SectorNumber(i) {
//							sct := model.NewSectorClaim(uint64(claimID), claim.Provider, claim.Client, claim.Data, claim.Size, claim.TermMin, claim.TermMax, claim.TermStart, claim.Sector, ts.Height())
//							res = append(res, sct)
//							continue
//						}
//					}
//
//					return nil
//				}); err != nil {
//					return fmt.Errorf("load sectorclaim info for %v at %v failed: %v", msg.To, ts.Height(), err)
//				}
//			}
//		}
//
//		// ExtendSectorExpiration || ExtendSectorExpiration2 续期
//		if EnableExtractMinerSector {
//			if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && (p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration || p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration2) {
//				fmt.Println(msg.Cid())
//				var extendSectors []bitfield.BitField
//				if p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration {
//					var params miner.ExtendSectorExpirationParams
//					if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//						return fmt.Errorf("unmarshal ExtendSectorExpirationParams for %v failed: %v", msg.Cid(), err)
//					}
//
//					for _, extension := range params.Extensions {
//						extendSectors = append(extendSectors, extension.Sectors)
//					}
//				} else if p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration2 {
//					var params miner.ExtendSectorExpiration2Params
//					if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//						return fmt.Errorf("unmarshal ExtendSectorExpiration2Params for %v failed: %v", msg.Cid(), err)
//					}
//
//					for _, extension := range params.Extensions {
//						extendSectors = append(extendSectors, extension.Sectors)
//					}
//				}
//
//				for i := range extendSectors {
//					sectorInfos, err := cmas.LoadSectors(&extendSectors[i])
//					if err != nil {
//						return fmt.Errorf("load sector infos for %v failed: %v", msg.To, err)
//					}
//
//					for _, info := range sectorInfos {
//						mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, false, ts.Height())
//						res = append(res, mst)
//					}
//				}
//			}
//		}
//
//		// ProveReplicaUpdates: replace proven sector
//		if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && (p.exec.Msg.Method == builtintypes.MethodsMiner.ProveReplicaUpdates || p.exec.Msg.Method == builtintypes.MethodsMiner.ProveReplicaUpdates2) {
//			var (
//				Deals            []abi.DealID
//				succeededSectors = bitfield.New()
//			)
//
//			if p.exec.Msg.Method == builtintypes.MethodsMiner.ProveReplicaUpdates {
//				var params miner.ProveReplicaUpdatesParams
//
//				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//					return fmt.Errorf("unmarshal ProveReplicaUpdatesParams for %v failed: %v", msg.Cid(), err)
//				}
//
//				if err := succeededSectors.UnmarshalCBOR(bytes.NewReader(p.exec.MsgRct.Return)); err != nil {
//					return fmt.Errorf("unmarshal ProveReplicaUpdates returns for %v failed: %v", msg.Cid(), err)
//				}
//			} else {
//				var params miner.ProveReplicaUpdatesParams2
//				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//					return fmt.Errorf("unmarshal ProveReplicaUpdatesParams2 for %v failed: %v", msg.Cid(), err)
//				}
//
//				if err := succeededSectors.UnmarshalCBOR(bytes.NewReader(p.exec.MsgRct.Return)); err != nil {
//					return fmt.Errorf("unmarshal ProveReplicaUpdates2 returns for %v failed: %v", msg.Cid(), err)
//				}
//			}
//
//			if EnableExtractMinerSector {
//				fmt.Println(msg.Cid())
//				var sectorInfos []*lminer.SectorOnChainInfo
//				if err := succeededSectors.ForEach(func(i uint64) error {
//					sectorInfo, err := cmas.GetSector(abi.SectorNumber(i))
//					if err != nil {
//						return fmt.Errorf("get sector %v for %v failed: %v", i, msg.To, err)
//					}
//
//					if sectorInfo != nil {
//						sectorInfos = append(sectorInfos, sectorInfo)
//					}
//
//					return nil
//				}); err != nil {
//					return fmt.Errorf("load sector info for %v at %v failed: %v", msg.To, ts.Height(), err)
//				}
//
//				for _, info := range sectorInfos {
//					mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, false, ts.Height())
//					res = append(res, mst)
//
//					Deals = append(Deals, info.DealIDs...)
//				}
//			}
//
//			if EnableExtractSectorClaim {
//				fmt.Println(msg.Cid())
//				var allocIDs []verifreg.AllocationId
//
//				state, err := cmkas.States()
//				if err != nil {
//					return fmt.Errorf("get market state failed: %v", err)
//				}
//
//				for _, deal := range Deals {
//					dealState, found, err := state.Get(deal)
//					if err != nil {
//						return fmt.Errorf("get dealstate failed: %v", err)
//					}
//
//					if found {
//						allocIDs = append(allocIDs, dealState.VerifiedClaim)
//					}
//				}
//
//				for _, allocID := range allocIDs {
//					claim, found, err := cvas.GetClaim(minerID, verifreg.ClaimId(allocID))
//					if err != nil {
//						return fmt.Errorf("get cliam for %v of %v failed: %v", allocID, minerID, err)
//					}
//
//					if found {
//						sct := model.NewSectorClaim(uint64(allocID), claim.Provider, claim.Client, claim.Data, claim.Size, claim.TermMin, claim.TermMax, claim.TermStart, claim.Sector, ts.Height())
//						res = append(res, sct)
//					}
//				}
//			}
//		}
//
//		// TerminateSectors: on-time terminate or early terminate
//		if EnableExtractMinerSector && p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && p.exec.Msg.Method == builtintypes.MethodsMiner.TerminateSectors {
//			fmt.Println(msg.Cid())
//			var params miner.TerminateSectorsParams
//			if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//				return fmt.Errorf("unmarshal TerminateSectorsParams for %v failed: %v", msg.Cid(), err)
//			}
//
//			var terminateSectors = bitfield.New()
//			for _, termination := range params.Terminations {
//				var err error
//				if terminateSectors, err = bitfield.MergeBitFields(terminateSectors, termination.Sectors); err != nil {
//					return fmt.Errorf("merge termination sectors failed for %v: %v", msg.To, err)
//				}
//			}
//
//			var sectorInfos []*lminer.SectorOnChainInfo
//			if err := terminateSectors.ForEach(func(i uint64) error {
//				sectorInfo, err := cmas.GetSector(abi.SectorNumber(i))
//				if err != nil {
//					// err is nil when not found
//					return fmt.Errorf("get sector %v for %v failed: %v", i, msg.To, err)
//				}
//
//				if sectorInfo != nil {
//					sectorInfos = append(sectorInfos, sectorInfo)
//				}
//
//				return nil
//			}); err != nil {
//				return fmt.Errorf("load sector info for %v at %v failed: %v", msg.To, ts.Height(), err)
//			}
//
//			for _, info := range sectorInfos {
//				mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, true, ts.Height())
//				res = append(res, mst)
//			}
//		}
//
//		// ExtendClaimTerms allows Partial failure
//		if EnableExtractSectorClaim && p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "verifiedregistry") && (p.exec.Msg.Method == builtintypes.MethodsVerifiedRegistry.ExtendClaimTerms || p.exec.Msg.Method == builtintypes.MethodsVerifiedRegistry.ExtendClaimTermsExported) {
//			fmt.Println(msg.Cid())
//			var params sverifreg.ExtendClaimTermsParams
//			if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
//				return fmt.Errorf("unmarshal ExtendClaimTermsParams for %v failed: %v", msg.Cid(), err)
//			}
//
//			for _, term := range params.Terms {
//				providerID, err := address.NewIDAddress(uint64(term.Provider))
//				if err != nil {
//					return fmt.Errorf("new id address for %v failed: %v", term.Provider, err)
//				}
//
//				claim, found, err := cvas.GetClaim(providerID, verifreg.ClaimId(term.ClaimId))
//				if err != nil || !found {
//					return fmt.Errorf("get claim for %v of %v failed: %v", providerID, term.ClaimId, err)
//				}
//
//				sct := model.NewSectorClaim(uint64(term.ClaimId), claim.Provider, claim.Client, claim.Data, claim.Size, claim.TermMin, claim.TermMax, claim.TermStart, claim.Sector, ts.Height())
//				res = append(res, sct)
//			}
//		}
//	}
//
//	fmt.Println("res", res)
//	return nil
//}
