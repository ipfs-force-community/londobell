

## 概念说明
正式库：用于保存bell程序所需的mongo代称
临时库： 用于保存tmpbell程序所需的mongo代称
冷库： 正式库数据量过大后，会将其设置为冷库，新启一个正式库保存bell数据

## 编译

此项目可分为两部分。calibnet与mainnet在编译时二进制不同，运行逻辑相同，后续不再单独说明    

- 链数据同步：
	链数据同步包含tmpbell,bell,orphanbell,fix-actormessage,共用同一bell二进制
  1. mainnet
  bell: `make build-bell`
  2. calibnet
	bell: `make build-bell-calib`
- 链数据后端服务: 
  后端服务分为agg和adpter，agg数据源为mongo存储，adapter为链的rpc服务的包装，数据源为链存储
  1. mainnet
  londobell-api-aggregators: `make build-aggregators`
  lotus-api-adapter:`build-adapter`
  2. calibnet
      londobell-api-aggregators: `make build-aggregators-calib`
      lotus-api-adapter:`build-adapter-calib`
## 运行

注意⚠️： bell运行需 零知识证明所需的相关数据，目录为： `/var/tmp/filecoin-proof-parameters`

Mongo索引⚠️：[index.js]()

参数配置说明：

- bell-repo： 为bell配置目录
- api-url,token为venus rpc配置

### TmpBell

实时同步最近的区块数据，放入mongo tmpbell库中(临时数据)
1. 初始化配置: 
`./bell --bell-repo=.tmpbell cfg init`
2. 设置配置参数:  name 为tmpbell数据库名
```bash
./bell --bell-repo=.tmpbell --api-url ws://113.240.65.28:13453/rpc/v0 --token"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.5OofPCBnQNaY-GFbj2bL_KfxsivwHEpPxNT3Y9TuDvw" segment update --name tmpbell --dsn-write-slice "mongodb://127.0.0.1:27017/tmpbell" --dsn-read "mongodb://127.0.0.1:27017/tmpbell" --child-hi bafy2bzacednj2zwkpsi3aivmtmncrwn6k3jy7heht6737sqax67ahgv66ch6k --child-lo bafy2bzacednj2zwkpsi3aivmtmncrwn6k3jy7heht6737sqax67ahgv66ch6k --set-active=true
```
3. 运行: --tmp: optional ，开启则为tmpbell，否则为bell
   `./bell --bell-repo=.tmpbell --api-url ws://113.240.65.28:13453/rpc/v0 --token "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.5OofPCBnQNaY-GFbj2bL_KfxsivwHEpPxNT3Y9TuDvw"  daemon run --tmp`

### OrphanBell

1. 运行
   `/root/londobell/bell --api-url=ws://192.168.10.3:1234/rpc/v0  orphan-daemon -dsn="" -name bell -ci 50`
   参数说明：
   - api-url：  lotus（venus需加token） rpc uri
   - dsn： 孤块存储mongo dsn，与bell保持一致
   - name： mongo dbname
   - ci: 孤块会实时同步链上的所有数据，然后在interval个区块间隔后，判断其是否为孤块，ci及为间隔的区块区间
### fix-actormessage
fix-actormessage 服务会修正正式库中的ActorMessage表中的to类型的消息所对应的ActorID

运行：

```bash
/root/londobell/bell fix-actormessage --dsn "" --name bell --nodeconfig /root/londobell/config.json --tick 2h
```
参数说明：

- dsn： 正式库dsn
- name: 正式库dbname
- nodeconfig： rpc的json配置文件
- tick：fix-actormessage运行间隔

### londobell-api-aggregators

1. 初始化配置
   `./londobell-api-aggregators --repo ../multi multiquery-cfg init`
2. 设置配置参数
   由于mongo数据量巨大，且一开始未存有全量数据，为支持翻页功能，为其单独建立索引，存入segment表中；

   ```bash
   ./londobell-api-aggregators --repo ../multi segment update --name segment --dsn-write "mongodb://127.0.0.1:27017/segment" --dsn-read "mongodb://127.0.0.1:27017/segment" --set-active=true
   ```

   - segment update: 设置segment 存储mongo
   - dsn-write ，--dsn-read： 读写数据库dsn配置
   - name ： segment数据库名
   - set-active： 将segment数据库名写以ipfs store的active key中，用于后续的索引更新
3. 更新segment索引
   ```bash
   ./londobell-api-aggregators --repo ../multi multiquery-cfg update --new-url "mongodb://192.168.1.225:27017" --new-name lsk_history --db-type 1 --nodeconfig ./config.json --force
   ```
   
   - multiquery-cfg update ：更新segment索引
   - new-url ： 待建立索引的mongo dsn
   - new-name ： 待建立索引的mongo dbname
   - db-type： mongo类型，0 临时库, 1 正式库, 2冷库 (default: 0)
   - nodeconfig： rpc的json配置文件
   - force： 是否替换原有的索引
   需将所有的冷库，正式库，临时库添加完毕，索引更新完后，会同时将配置写入config中，以供agg运行时读取
   
4. 运行
   ```bash
   ./londobell-api-aggregators --repo=../multi daemon --port 2345 --nodeconfig=./config.json --RPCListen /ip4/127.0.0.1/tcp/22347
   ```

   - port: agg 端口设置
   - nodeconfig：rpc的json配置文件
   - RPCListen： metrics 相关的rpc端口配置
#### lotus-api-adapter
运行：

`/root/londobell/lotus-api-adapter daemon --bell-repo=../bellstore --port 12345 --nodeconfig=/root/londobell/config.json`    

- port : adapter api监听端口
- nodeconfig：rpc json配置文件
- bell-repo： repo目录，未有实际作业

## 架构设计

### 链数据同步整体设计![alt text](./chain-arch.png)

<img src="./chain-arch.png" alt="image-20240206114002965" style="zoom: 50%;" />

#### bell 运行流程

<img src="./bell-process.png" alt="image-20240206171645400" style="zoom:67%;" />
bell从rpc处监听chain head，获取对应的tipset，并通过chain store load ExtractLinkedTipSets,然后通过Extract存入Mongo



### 后端服务整体设计



<img src="./backend-arch.png" alt="image-20240207141614744" style="zoom:67%;" />

