# Londobell 随 Filecoin 网络升级的检查项清单
需要检查确认的事项主要包含以下方面：
- 是否正确引入新的 specs-actors 版本
- 新 specs-actors 的迁移项范围
- 上一版本的 Extractor 逻辑是否依然适用
- 从外部(specs-actors、lotus 等)代码中摘取的逻辑是否依然适用，摘取的逻辑是否存在被转换成可见的函数的情况

基于此，我们拟定清单如下：
- [ ] 执行 `./tool/scripts/upgrade-lotus.sh <target version>` 将本库的lotus升级为指定版本
- [ ] 执行 `./tool/scripts/submodule-check.sh <target version>` 检查ffi是否更新
- [ ] 执行 `make gen-extractor`，沿用之前的逻辑生成出新版本的 extractor 代码 
- [ ] 尝试执行 `make build-bell` 和 `make build-bell-calib`，并解决可能出现的编译器错误，此时的编译错误通常由以下 lotus 内部的变化导致：
  - 重命名
  - 代码结构调整
  - 函数签名变化

- [ ] 执行 `make gen-diff` 观察:
  - 各 State 定义的变化情况
  - 各 Method 的入参和返回所包含的数据类型变化情况

- [ ] 检查由 specs-actors 中直接摘录的代码逻辑是否依然生效，具体做法为：
  - 根据关键字 `VERCHECK` 查找标记处的代码段
  - 对比最新版本中的逻辑代码与上一版本的逻辑是否有变化，根据变化情况完成代码改动

- [ ] 执行`make gen-index`, `make gen-model` 更新索引和示例

- [ ] 解决完编译时问题后，尝试启动一个 `racailum` 实例，并解决可能出现的依赖注入错误

- [ ] 将新版本投入 `calibnet` 测试并观察测试结果


londobell-api随 Filecoin 网络升级的检查项清单
需要检查确认的事项主要包含以下方面：
- 是否正确引入新的 specs-actors 版本
- 新 specs-actors 的迁移项范围
- 从外部(specs-actors、lotus 等)代码中摘取的逻辑是否依然适用，摘取的逻辑是否存在被转换成可见的函数的情况

基于此，我们拟定清单如下：
- [ ] 执行 `make gen-types`, 沿用之前的逻辑生成出新版本的 types 代码
- [ ] 尝试执行 `make build-adapter`，并解决可能出现的编译器错误，此时的编译错误通常由以下 lotus 内部的变化导致：
  - 重命名
  - 代码结构调整
  - 函数签名变化
- [ ] 尝试执行 `make build-aggregators`，并解决可能出现的编译器错误


### Others

#### 检查`github.com/filecoin-project/lotus/node/` build变更，根据变更调整`dep`部分代码

1. calibnet 网络：`~/go/pkg/mod/github.com/filecoin-project/lotus\@v1.30.0/build/buildconstants/calibnet.go` 注释 `DrandSchedule`

```go
var DrandSchedule = map[abi.ChainEpoch]DrandEnum{
        // 0:                    DrandMainnet,
        UpgradePhoenixHeight: DrandQuicknet,
}
```

1. 主网：`~/go/pkg/mod/github.com/filecoin-project/lotus\@v1.30.0/build/buildconstants/params_mainnet.go` 注释 `DrandSchedule`

  ```go
  var DrandSchedule = map[abi.ChainEpoch]DrandEnum{
          //0:                    DrandIncentinet,
          // UpgradeSmokeHeight:   DrandMainnet,
          UpgradePhoenixHeight: DrandQuicknet,
  }
```

#### 检查 method

检查本次升级中是否有新增 `method`，若有则调整 `cmd/londobell-api/util/util.go` 中 `AllMethodList`。
