package main

//func Init() {
//	//fmt.Println(DBStateManager)
//	fullnode.API = fullnode.NewAppropriateAPI([]util.Node{{URL: "<LOTUS_RPC_URL>"}})
//	err := fullnode.API.Choose(context.TODO())
//	if err != nil {
//		return
//	}
//
//	_, err = dix.New(
//		context.TODO(),
//		dep.MultiQuery(context.TODO(), &multiquery.DBStateManager),
//		MockRepoPath(),
//		//dep.InjectFullNode(cctx),
//	)
//	if err != nil {
//		fmt.Println("stopper", err)
//		return
//	}
//}
//
//func MockRepoPath() dix.Option {
//	return dix.Override(new(dep.RepoPath), func() (dep.RepoPath, error) {
//		return dep.RepoPath("/home/user/.multi"), nil
//	})
//}
//
//func MockConfig() error {
//	cfgPath := dep.ConfigFilePath("/home/user/.multi")
//
//	err := os.MkdirAll(filepath.Dir(cfgPath), 0755)
//	if err != nil {
//		return fmt.Errorf("MkdirAll for %s: %w", cfgPath, err)
//	}
//
//	cfg := MockDefaultConfig()
//	content, err := config.ConfigComment(cfg)
//	if err != nil {
//		return fmt.Errorf("marshal default config: %w", err)
//	}
//
//	err = ioutil.WriteFile(cfgPath, content, 0644)
//	if err != nil {
//		return fmt.Errorf("write config file: %w", err)
//	}
//
//	return nil
//}
//
//func MockDefaultConfig() config2.Config {
//	colds := make([]config2.DB, 0)
//	colds = append(colds, config2.NewDB("mongodb://guest:read-only@<MONGO_HOST>:27017/bell", "bell"))
//	return config2.Config{
//		Colds:  colds,
//		Formal: config2.NewDB("mongodb://guest:read-only@<MONGO_HOST>:27017/bell", "bell"),
//		Tmp:    config2.NewDB("mongodb://127.0.0.1:27017/tmpbell", "tmpbell"),
//	}
//}
//
//func TestRefresh(t *testing.T) {
//	Init()
//	DBStateManager.DBCfg.Cfg = MockDefaultConfig()
//	err := DBStateManager.LoadDBCollectionsMap(context.TODO())
//	require.NoError(t, err, "LoadDBCollectionsMap failed")
//	formal := DBStateManager.GetFormalCfg()
//	cols, ok := DBStateManager.GetDBCollections(formal.Url())
//	require.Equal(t, true, ok)
//
//	dbState := &DataBaseState{
//		StartEpoch:                        1960559,
//		EndEpoch:                          1960619, //2317676, // formal每次从finalHeight拿
//		NextEpochForBlockMsgsCount:        1960559,
//		NextEpochForBlockMsgsByMethodName: 1960559,
//		NextEpochForActorMsgsByMethodName: 1960559,
//		NextEpochForActorMsgsCount:        1960559,
//		NextEpochForActorTransfersCount:   1960559,
//		NextEpochForMinedMsgs:             1960559,
//		NextEpochForTransfersLargeAmount:  1960559,
//		BlockMsgsCount:                    int64(0),
//		BlockMsgsByMethodNameMap:          make(map[string]int64),
//		ActorMsgsByMethodNameMap:          make(map[string]map[string]int64),
//		ActorMsgsCountMap:                 make(map[string]int64),
//		ActorTransfersCountMap:            make(map[string]int64),
//		MinedMsgsMap:                      make(map[string]int64),
//		TransfersLargeAmountCount:         int64(0),
//	}
//
//	//dbState := &DataBaseState{
//	//	Name:                              "bell",
//	//	StartEpoch:                        2500422,
//	//	EndEpoch:                          2503302,
//	//	NextEpochForBlockMsgsCount:        2500422,
//	//	NextEpochForBlockMsgsByMethodName: 2500422,
//	//	NextEpochForActorMsgsByMethodName: 2500422,
//	//	NextEpochForActorMsgsCount:        2500422,
//	//	NextEpochForActorTransfersCount:   2500422,
//	//	NextEpochForMinedMsgs:             2500422,
//	//	NextEpochForTransfersLargeAmount:  2500422,
//	//	BlockMsgsCount:                    int64(0),
//	//	BlockMsgsByMethodNameMap:          make(map[string]int64),
//	//	ActorMsgsByMethodNameMap:          make(map[string]map[string]int64),
//	//	ActorMsgsCountMap:                 make(map[string]int64),
//	//	ActorTransfersCountMap:            make(map[string]int64),
//	//	MinedMsgsMap:                      make(map[string]int64),
//	//	TransfersLargeAmountCount:         int64(0),
//	//}
//
//	// Epoch_1_Depth_1_Msg.From_1索引创建后速度提升
//	start := time.Now()
//	err = RefreshBlockMsgs(context.TODO(), dbState, cols) //1m0.859942454s //106.80727ms
//	require.NoError(t, err, "RefreshBlockMsgs failed")
//	fmt.Println("refreshBlockMsgs elapsed: ", time.Now().Sub(start).String())
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	start = time.Now()
//	err = RefreshBlockMsgsByMethodName(context.TODO(), dbState, cols) //8m21.140332002s //1.174092407s
//	require.NoError(t, err, "RefreshBlockMsgsByMethodName failed")
//	fmt.Println("refreshBlockMsgs elapsed: ", time.Now().Sub(start).String())
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	start = time.Now()
//	err = RefreshActorMsgsByMethodName(context.TODO(), dbState, cols)
//	require.NoError(t, err, "RefreshActorMsgsByMethodName failed")
//	fmt.Println("refreshActorMsgsByMethodName elapsed: ", time.Now().Sub(start).String()) //45.228966131s
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	start = time.Now()
//	err = RefreshActorMsgs(context.TODO(), dbState, cols)
//	require.NoError(t, err, "refreshActorMsgs failed")
//	fmt.Println("refreshActorMsgs elapsed: ", time.Now().Sub(start).String()) // 246.29034ms
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	start = time.Now()
//	err = RefreshActorTransferMsgs(context.TODO(), dbState, cols)
//	require.NoError(t, err, "refreshActorTransferMsgs failed")
//	fmt.Println("refreshActorTransferMsgs elapsed: ", time.Now().Sub(start).String()) //12.202645925s
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	start = time.Now()
//	err = RefreshMinedMsgsMaps(context.TODO(), dbState, cols)
//	require.NoError(t, err, "refreshMinedMsgsMaps failed")
//	fmt.Println("refreshMinedMsgsMaps elapsed: ", time.Now().Sub(start).String()) //15.128695ms
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	start = time.Now()
//	err = RefreshTransfersForLargeAmount(context.TODO(), dbState, cols)
//	require.NoError(t, err, "RefreshTransfersForLargeAmount failed")
//	fmt.Println("refreshTransfersForLargeAmount elapsed: ", time.Now().Sub(start).String()) //5.732332695s
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	err = DBStateManager.Stm.SetDataBaseState(formal.Url(), *dbState)
//	require.NoError(t, err, "SetDataBaseState failed")
//}
//
//func TestLoadDataBase(t *testing.T) {
//	//TestRefresh(t)
//	Init()
//	dbState, found, err := DBStateManager.Stm.LoadDataBaseState("mongodb://guest:read-only@<MONGO_HOST>:27017/bell")
//	require.NoError(t, err, "LoadDataBaseState failed")
//	require.Equal(t, true, found)
//
//	fmt.Printf("dbState: %+v\n", dbState)
//
//	file, err := os.OpenFile("/home/user/londobell/cmd/londobell-api/aggregators/bell.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
//	require.NoError(t, err, "open failed")
//	defer file.Close()
//	_, err = io.WriteString(file, fmt.Sprintf("startEpoch: %v, endEpoch: %v, NextEpochForBlockMsgsCount: %v, BlockMsgsCount: %v\n", dbState.StartEpoch, dbState.EndEpoch, dbState.NextEpochForBlockMsgsCount, dbState.BlockMsgsCount))
//	require.NoError(t, err, "write failed")
//}
