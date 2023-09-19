#### 一、aggregators
aggregators组件从londobell数据库读取落库数据，为filscan提供历史数据相关接口。
1. dbstate   
londobell数据库一般4个月左右进行分库，来避免单库数据量过大。为了优化分库数据库的查询速度，新增存放分库数据库状态(dbstate)的repo，即将分库数据库的数据按一段高度范围进行分段统计，存入该repo中。
以下是分段状态repo的相关命令：
- 初始化dbstate repo  
```
./londobell-api-aggregators --repo=~/.multi multiquery-cfg init
```
该命令会在~/.multi目录下新建配置文件config,config内容如下：
```
Colds = []
LastModifyTime = 1695094272
BatchSegmentInsertLimit = 16

[Formal]
  URL = ""
  DBName = ""

[Tmp]
  URL = ""
  DBName = ""
```
分库数据库分为三类：Colds(冷库)、Formal(正式库)、Tmp(临时库)。冷库可以有多个，正式库和临时库只有一个。
LastModifyTime记录该config被修改的时间戳，可以直接在config文件中更改临时库的配置，系统会每隔30s监听配置文件改动并生效。
BatchSegmentInsertLimit指定并发对数据库进行分段状态处理时限制的并发数，默认是16。

- 指定存放分段状态(即dbstate)的数据库
```
./londobell-api-aggregators --repo=~/.multi segment update --name segment --dsn-write "mongodb://segment:segment@127.0.0.1:27017/segment" --dsn-read "mongodb://guest:read-only@127.0.0.1:27017/segment" --set-active=true
```
该命令指定存放分库数据库分段状态的数据库，会在~/.multi/state下存放segment数据库的信息。--name指定数据库名；--dsn-write指定数据库可写dsn；--dsn-read指定数据库只读dsn；--set-active指定是否激活。

- 查询分段数据库的信息  
在线查询：
```
./londobell-api-aggregators --repo=~/.multi segment show --name segment --RPCListen /ip4/127.0.0.1/tcp/12345 --local=false
```

离线查询：
```
./londobell-api-aggregators --repo=~/.multi segment show --name segment --local=true
```
当aggregators启动时，可以通过rpc（在线）查询其分段数据库信息，或在aggregators终止时直接访问repo（离线）查询其分段数据库信息（注意：repo不能同时被两个进程访问）。--name指定查询分段数据库名；--RPCListen指定aggregators启动时指定的监听地址；--local指定在线(false)或离线(true)方式

- 新增数据库
```
./londobell-api-aggregators --repo=~/.multi multiquery-cfg update --new-url "mongodb://guest:read-only@127.0.0.1:27017/bell" --new-name bell --db-type 2 --nodeconfig ./config.json
```
该命令为新增冷库bell，会在~/.multi/config中添加该冷库配置，且对该数据库进行分段状态统计并存储在segment数据库中。--db-type可填0(Tmp)、1(Formal)、2(Colds); --nodeconfig存放连接远程节点（lotus or venus）的配置，内容如下：
```
[
        {
                "node":"ws://127.0.0.1:3453/rpc/v0",
                "token":"abc"
        }
]
```
可连多个远程节点，系统会每隔15s轮询当前同步在主链最高高度的节点连接。

- Formal库转存为冷库
```
./londobell-api-aggregators --repo=~/.multi dbstate archive --formal-url "mongodb://guest:read-only@127.0.0.1:27017/bell" --formal-name bell --cold-url "mongodb://guest:read-only@127.0.0.1:27017/cold" --cold-name cold
```
该命令会删除segment数据库中Formal的记录，同时添加冷库的状态；~/.multi/config会删掉Formal的配置，同时添加冷库的配置。

- 更新dbstate
```
./londobell-api-aggregators --repo=../multi dbstate update --url "mongodb://guest:read-only@127.0.0.1:27017/bell" --name bell --utype 0
```
该命令会更新指定数据库的utype，截止高度为指定数据库的FinalHeight。
--utype可填0(块消息分段)、1(根据方法名筛选块消息分段)、2(根据方法名筛选区块消息分段)、3(根据actorID筛选块消息分段)、4(根据actorID和方法名筛选块消息分段)、5(根据actorID筛选转账消息分段)、6(根据actorID筛选事件分段)、7(根据minerID筛选出块消息分段)、8(大额转账消息分段)、9(订单列表分段)、10(根据ID地址筛选订单列表分段)、11(tipset分段)、12(所有状态分段)

- 删除dbstate
```
./londobell-api-aggregators --repo=~/.multi dbstate delete --url "mongodb://guest:read-only@127.0.0.1:27017/bell" --name bell --dtype 12 --db-type 2
```
该命令会删除指定数据库的dtype。dtype可填值和上述utype相同，当--dtype指定为12，即删除所有分段状态时，会删除对应类型db-type的配置

2. 启动aggregators组件  
```
nohup ./londobell-api-aggregators --repo=~/.multi daemon --port 1234 --nodeconfig=./config.json --RPCListen /ip4/127.0.0.1/tcp/12345 >>agg.log 2>&1 &
```
该命令后台启动aggregators组件。--repo指定存放dbstate的目录；--port指定aggregators监听的端口；--nodeconfig同上存放连接节点信息；--RPCListen指定运行远程查询分段数据库状态的地址

#### 二、 adapter
adapter组件通过连接远程节点(lotus or venus)来读取实时状态数据，为filscan提供实时数据
1. 启动adapter组件
```
nohup ./lotus-api-adapter daemon --port 12345 --nodeconfig ./config.json --bell-repo ~/.bell >>adapter.log 2>&1 &
```
该命令后台启动adapter组件。--port指定监听的端口；--nodeconfig同上；--bell-repo指定adapter依赖注入节点store的目录，不填则默认为~/.bell
