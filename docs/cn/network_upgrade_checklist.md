# Londobell 随 Filecoin 网络升级的检查项清单
需要检查确认的事项主要包含以下方面：
- 是否正确引入新的 specs-actors 版本
- 新 specs-actors 的迁移项范围
- 上一版本的 Extractor 逻辑是否依然适用
- 从外部(specs-actors、lotus 等)代码中摘取的逻辑是否依然适用，摘取的逻辑是否存在被转换成可见的函数的情况

基于此，我们拟定清单如下：
- [] 执行 `./tool/scripts/upgrade-lotus.sh <target version>` 将本库的lotus升级为指定版本
- [] 尝试执行 `make build-bell`，并解决可能出现的编译器错误，此时的编译错误通常由以下 lotus 内部的变化导致：
  - 重命名
  - 代码结构调整
  - 函数签名变化

- [] 尝试启动一个 `racailum` 实例，并解决可能出现的依赖注入错误

- [] 在 `racailum/segment/actor/specs.go` 文件中引入新的 specs-actors 版本

- [] 执行 `make gen-diff` 观察:
  - 各 State 定义的变化情况
  - 各 Method 的入参和返回所包含的数据类型变化情况

- [] 检查由 specs-actors 中直接摘录的代码逻辑是否依然生效，包含：
  - [] empty miner state 的构造和判断逻辑，即 `racailum/segment/extract/actorstate.templates/miner_state_common.one.template` 中的逻辑，完成：
    - 检查 ``MinerActor.ConstructState`` 方法的逻辑变化情况
	- 为模板中的 `newEmptyMinerState` 函数增加对应版本的代码段注释
