# bell 缺块巡检方案（cali）

> **本方案已按上机实测改写（2026-10-11）。** 初版设计的「离线抽取器巡检」经验证**不可行**，
> 死路证据见第一节。现行方案改为**读 bell 自己的日志**——零成本、不占任何服务、已实证跑通。

## 一、被否决的方案（留档，避免重走）

初版设想：每天用 `bell offline extractor` 干跑最近 N 天，报 `could not find` 的高度即真洞。
上机实测三条全部失败：

| 尝试 | 参数 | 实测结果 |
|---|---|---|
| 本地模式 | `--local=true`（默认） | `panic: lock repo failed: repo is already locked (process is already running)` — bell 正持 repo 锁 |
| 远程模式 | `--local=false` | `cli error: starting node: … resource temporarily unavailable` — mongo 连接打满 |
| 只读保证 | `--writableOffline=false` | 确认只读（源码 `dep.InjectWritableOffline`），但救不了上面两条 |

**根因**：cali 节点机是 2 核 16G，跑不动「巡检 + bell + syncer + lotus + mongo 客户端」这一整套。
**附带更正**：初版文档写「`--local=false` 可绕开会话锁」是错的——昨晚能跑通靠的是**另建独立 repo 副本**，
不是参数本身；该现场（独立 repo）已被清理，无法复现。

### 另一条被否决的判据：用 `ChainHasObj` 查状态根

实测扫描最近 24 小时 1000 个 tipset 状态根，**缺失 1000 个（100%）**，而那 1000 个高度
数据完整、站点正常服务。原因：`/root/.lotus/config.toml` 里 `EnableSplitstore = true`，
冷状态树被**按设计**裁掉。

⇒ **不能用「状态根在不在本地」判缺块**，会得到 100% 假阳性。
（初版文档据此误报过「大面积缺失、故障前兆」，已撤回。）

## 二、现行方案：读 bell 日志

**原理**：bell 每轮都从 `段边界+1` 抽到链头，它撞到的洞**一定会留在自己的日志里**。
不需要额外跑任何东西，也不影响、不占用任何服务。

### 提取命令（2026-10-11 09:14 实测跑通）

```bash
# 1) 所有曾被 bell 报错的洞高度（跨全部轮转文件）
for f in bell.log bell.log.1 bell.log.2 bell.log.3; do
  [ -r "$f" ] || continue
  grep -a "extract job failed after 2 attempt" "$f" \
    | grep -aoE 'tipset.Height: [0-9]+' | grep -oE '[0-9]+'
done | sort -n | uniq -c

# 2) 缺失的对象 CID（排除 events 这类可容忍的缺失）
for f in bell.log bell.log.1 bell.log.2 bell.log.3; do
  [ -r "$f" ] || continue
  grep -a "could not find" "$f" | grep -av "get events failed" \
    | grep -aoE 'could not find [a-z0-9]+' | awk '{print $NF}'
done | sort -u
```

**实测产出**：11 个洞高度、13 个 distinct CID。其中 `4138627` 被报 25 次（反复撞），其余各 1~2 次。

### 判据：哪个洞是真数据空洞

**唯一判据是「洞高度是否在已发布水位之下」**，不是「本地有没有状态根」。

```bash
FH=$(curl -s -X POST http://172.31.41.248:1237/aggregators/final_height \
      -H 'content-type: application/json' -d '{}' | grep -oE '"Epoch":[0-9]+')
# 洞高度 < FH ⇒ 已发布区间内有洞 ⇒ 可能已对外暴露，必须核 PG 并回填
# 洞高度 > FH ⇒ 尚未发布，补块即可，不对外暴露
```

实测：11 个洞**全部低于当时水位 4145148**。但逐高度查 PG 确认数据都在
（`actor_actions` 12~17 行、`builtin_actor_states` 各 2、`miner_rewards` 2~4 行）
⇒ **补块发生在发布之前，未产生对外空洞**。

### 落点：不要新建，扩现有的 logwatch（重要更正）

初版方案写「新建一个 cron 做巡检」——**这是重复造轮子，已否决**。
实测发现 cali bell 机上已有现成机制：

```
root crontab（每 5 分钟）：
  /opt/filscan-logwatch/logwatch.sh   → 扫 bell.log 的 extract job failed / could not find / failed to persist
                                         推成指标 filscan_log_error_events{pattern=...}（含增量与累计）
  /opt/filscan-heal/healwatch.sh      → 撞洞后的自动补齐
```

⇒ **「发现洞」这件事已经在做了**，且带 state 偏移（只统计新增、轮转时基线后移、计数单调不假增）。

**真正缺的是判据，不是发现**：现有计数器回答「有没有洞」，但不回答
**「这个洞有没有已经对外发布（= 真数据空洞）」**。这才是 2026-10-10 那次
31.5 小时事故里最需要立刻知道、而当时没人能立刻答出来的问题。

**建议**：在 `logwatch.sh` 里补一条旁路（不改它现有指标口径与名字）：

```
当新增 could_not_find > 0 时：
  取洞高度（从 bell.log 尾部窗口提 tipset.Height）
  取当前 final_height（curl agg:1237）
  若存在 洞高度 < final_height：
      推一条独立指标 filscan_below_watermark_holes{source="calibnet-bell"} <个数>
      （PG 行数核对放在人介入时做，不在 5 分钟脚本里挂长查询）
```

这样告警语义从「日志里有错误形态」升级为「**已发布区间内可能缺数据**」，后者才是要盯的。

**若判定复用不可行**（例如不想动现有脚本），退路是新建独立脚本，但**必须先说明为什么不能在现有脚本里做**。

## 三、与 heal 的职责分界

| | heal watch（现状） | 本文案（日志巡检） |
|---|---|---|
| 触发 | bell 撞洞**之后** | 每日定时 |
| 作用 | 补齐缺失的块 | **发现**洞 + 判断是否已对外暴露 |
| 是否改生产 | 是（写 badger） | 否（纯读日志 + 查 PG） |

**两者是兜底与监控的关系**，heal 保留；本方案**不替换 heal、不改其写回逻辑**。

## 四、明确不做的事（及原因）

| 不做 | 原因 |
|---|---|
| 离线抽取器全量巡检 | 已实测不可行（第一节） |
| 扫状态根 | 100% 假阳性（splitstore 设计） |
| 扫「最近 14 天」 | 状态保留窗口仅 36–40 小时，超窗口的洞 bell 自己也补不了，扫了无用 |
| **新建独立巡检 cron** | **已有 `logwatch.sh` 在做同一件事，重复造轮子**；应扩它而不是另起一套 |
| 改 bell 让它不撞洞 | 属 bell 侧批次语义改造，见 `batch_failure_semantics.md`，单独评估 |

## 五、验收判据

1. **判别力**：对已知已补齐的洞高度（如 4141401），巡检应报「曾出现、已补回且 PG 有数据」，
   而不是「当前仍缺」。
2. **零副作用**：全程不写任何存储、不占 repo 锁、不影响 bell 的 `done-parts` 涨幅。
3. **告警准确性**：只在「洞低于水位且 PG 无数据」时报红色，其余为信息级。

## 六、已知陷阱

1. **`grep -c` 在 `.gz` 上恒为 0**——压缩日志必须用 `zgrep`，否则会把「历史有洞」误判成「历史干净」。
   （本次比基线时踩过：把 10-09 的 153,600 条慢查询读成 0。）
2. **双层 ssh 下别内联复杂引号**，一律落脚本文件再跑。
3. **脚本的 `/tmp` 要落在目标机**：经跳板机时用 `cat > /tmp/x.sh` 显式写入第二跳。
