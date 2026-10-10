# bell 批次的失败语义：为什么「一个小故障」会放大成「整段停摆」

本文记录 bell（londobell 的链数据写入侧）在批次失败时的实际行为、两次线上事故的实测数据、
以及为什么「坏一个 part」会作废整批。写这份文档是因为 2026-10-10 的 calibnet 事故里，
一个 6 个块的缺失把站点拖停 31.5 小时，而这个放大效应此前在仓内没有任何记录。

## 一、事实层：提交是批级原子的，且没有部分成功

`racailum/segment/extract.go` 的 `Segment.Extract`：

- 区间 = 「段边界 + 1 → 链头」（由 `Extract` 的调用方传入，见 `segment.go:289` / `ra.go:171`）；
- 区间切 part（`TipSetPartSizeLimit`，主网实测 16），part 内按 `TipSetJobLimit`（主网实测 4）并发抽 tipset；
- **任一 part 连续失败 `attempt` 次（实测 2 次）⇒ `extractJobWithTimeout` 返回 error ⇒ 整批失败**；
- `FinalHeight`（对外发布水位）**只在整批 persist 成功后写入一次**。

`extractJobWithTimeout` 的注释明确了这是有意设计：

> 上层（Segment.Extract → RaCailum.Run 循环）会在下一个 tipset 到来时按已提交的 final_height 整段重跑，
> 因此这里不需要更复杂的恢复逻辑。

注意两点容易被误读的：

1. **「整批失败」不等于「已写入的数据被删掉」**。已 persist 的 part 其文档留在 mongo 里，
   下轮重跑会覆盖（写入侧有 `epochValue >= existingEpoch` 的保护，见 `racailum/segment/persist.go`）。
   所以「站点停了」和「数据没写」是两种现象，别用后者解释前者。
2. **重试区间的上沿随链头增长**，起点固定。这是它不可自愈的结构性原因：落后越多 ⇒ 区间越大 ⇒
   越容易再次超时 ⇒ 区间更大。

## 二、实测：放大倍率有多大

### 案例 A：calibnet，2026-10-10（块缺失型）

| 项 | 值 |
|---|---|
| 站点停摆 | 31.5 小时（final_height 4138552 → 4144044，落后峰值 5829 高度） |
| 根因 | 本地 lotus blockstore 缺 6 个状态树节点（4140561 / 4141154 / 4141280 / 4141349 / 4141401 / 4141444） |
| 缺失块总量 | 84 个小块（每个洞 14 块，heal 每轮 `written=14`） |
| 被作废的批 | `[4140557, 4144044]` = **3487 个高度** |
| 已成功后被拖着作废的部分 | 826 个高度（约 24%） |

**84 个块的问题 → 3487 个高度作废 → 31.5 小时停摆。** 而 heal 补一个洞只要 40 秒
（`probe done missing=14` → `HEAL_OK written=14`）。也就是说：
**每个洞的修复成本是 40 秒，但每个洞的等待成本是约 1 小时（一次整批重跑）。**

更糟的是 heal 本身会打断 bell：healwatch 每次修洞都要
`supervisorctl stop lotus bell statebell tmpbell` → 写入 → 重启，
本次触发 6 次，每次都让 bell 刚起步的批次清零。**修洞的动作本身在延长故障。**

### 案例 B：mainnet，2026-10-09 / 10-11（落库超时型）

同源不同相：不是缺块，是 mongo 写入打满超时。

- `failed to persist tipset: error occurs in async persist: connection(<mongo-host>:27017) incomplete read of message header: context deadline exceeded`
- 2026-10-09：2053 高度 = 129 个 part 的大批，两次尝试各跑满约 2 小时，**都在 `done-parts=128/129` 时被作废**；
  16.6 小时内 32 次全败（起点固定 6438943、上沿 6439170→6439797 一路涨）。
- 2026-10-11：同一形态复发（03:22 / 04:38），区间 `[6444712, 6444952]` 重抽 3 轮，每轮 1h16m~1h22m。

**注意「128/129」这个数字**：99.2% 的工作已经做完，只因最后一个 part 失败而全部作废。

## 三、已做的措施（含一处需要更新的旧结论）

`racailum/segment/options.go` 的 `Persist.WaitTimeout` 已由 10 分钟改为 **60 分钟**
（commit `e419c1ad`，2026-10-09；mainnet 线上二进制实测 `vcs.revision=e419c1ad`，已生效）。
它包住两处：`extract.go` 的 `insertManyWithTimeout`（每次 insert）与 `waitAsyncPersist`（等整批落库）。

这条修法让「大批有机会跑完」，但 2026-10-11 的复发表明：**60 分钟也可能被更大的批打穿**。
根因在 mongo 侧的读放大——单批 4096 条 `bell.ActorBalance` insert 要 30~40 秒、
其中 `timeReadingMicros` 占 29~38 秒（每条约 89 KB 随机读），`cache_size` 15 GB 对 1.4 TB 数据。
调超时是让大批有机会跑完，不是治读放大。

## 四、能力不对等：消费侧早就有「跳过 + 留账」，制造侧没有

这是本文最重要的发现，也是后续改动的依据。

| 能力 | filscan_backend（同步器，数据消费侧） | londobell（bell，数据制造侧） |
|---|---|---|
| 连续失败后跳过 | **有**：`data_error_threshold`（默认 5）+ `unrecoverable_error_threshold` | **无** |
| 整段跳过 | **有**：`state_gap_jump`（缺口 > 1000 时跳到链头−margin） | **无** |
| 跳过必须留痕 | **有**：`chain.sync_skipped_epochs`（migration 33 + 34，记 `error_class`/`skipped_from`/`skipped_to`）；**写不进台账就不允许跳过**，继续按原逻辑重试 | **无** |
| 回填缺口的工具 | 有（带 `GapScan` 的 `chain-miner` 回放、`--epochs-file` 离散清单模式等） | **无**（heal 只补 blockstore 缺块，不补「已登记的坏 part」） |

全仓搜索 `skip.*(epoch|part|tipset) | skipped | gap_jump | tolerate` 在 racailum 下 **0 命中**。

⇒ **cali 的 6 个洞之所以滚成 31.5 小时，根因不是「批次原子性」这个设计选择，
而是 bell 这一侧连「坏 part 单独记账、其余照常提交」的能力都没有。**
上游消费侧早就有这套机制并且天天在用，制造数据的一侧反而是空的。

## 五、设计方向：把失败的影响范围从「整批」缩到「那一个 part」

**目标不是弱化批次提交**（正确性优先的原子提交保留），而是让「坏 part」不再拖着好 part 一起作废。

### 5.1 失败分级：先分清是哪一类错再决定处不处断

沿用上游 `error_class` 的思路分三类，**处断方式完全不同**：

| 类别 | 判据 | 处置 |
|---|---|---|
| 数据级 | 该 part 内容有问题（如 exec-trace 解不出、毒高度） | 登记 → 跳过 → 事后回填 |
| 状态级（不可恢复） | 节点侧状态已裁掉，再等也不会好 | 登记 → 跳过 → 借归档源回填或长期留账 |
| 传输级 | 网络/超时/连接被打断 | **不跳过**，保持无限重试（现状） |

关键分界：**同一错误连续出现 N 次且高度/内容固定 ⇒ 数据级（跳过有意义）；每次失败的位置都在变 ⇒ 传输/资源级（重试有意义）。**

### 5.2 台账：一张表，语义照抄上游

新增 `chain.sync_skipped_parts`（或等价命名，bridge 到既有命名风格）：

- `(skipped_from, skipped_to)` 区间、`error_class`、`cid`（触发失败的那个根 CID）、
  `attempts`、`first_seen`、`last_seen`、`healed_at`（回填成功后回填该列）；
- **唯一键 `(skipped_from, skipped_to)`** ⇒ 同一区间重复失败只更新 `last_seen`/`attempts`，不插新行；
- **写不进台账就不允许跳过** —— 照上游那条铁律，否则会变成静默跳洞。

### 5.3 提交粒度：part 级 final_height

`FinalHeight` 当前只在整批成功后写。改为**按已成功的连续前缀推进**：
`SaveFinalHeight` 的入参从「整批」变成「本批已成功的最大连续 part 上界」。
这样站点其余数据能立刻恢复，只剩被登记的洞区还是旧的——**用户看到的是「大部分新、一小段旧」，而不是「全部旧 31 小时」**。

### 5.4 回填：闭环必须接上，否则就是「制造了洞」

被登记的坏 part 需要一条自动回填路径，否则新机制等于把「整批停摆」换成「静默数据洞」（这正是用户明确否掉过的形态）。

现有可复用的两件：
- **heal**（部署方自运维工具，不在本仓）已经从公网取回缺失块并写回本地 blockstore，验证过 40 秒/洞、0 失败；
  它是补 blockstore 缺块的前置手段，本仓需要补的是「已登记的坏 part」这一层。
- 回填成功后，`chain.sync_skipped_parts` 里对应的 `healed_at` 被置上，同时 **把该区间的数据重抽提交**。

⇒ 顺序是「登记 → 提交好部分 → heal/回填 → 补齐坏部分 → 台账销项」。

### 5.5 重试退避

bell 撞洞失败后**立刻从头重来**（实测每 55~90 分钟一轮，每轮都从 `段边界+1` 重抽）。
应改为指数退避（如 5min → 10min → 20min，上限 30min），
**理由：cali 的每一轮都在已经知道会失败的高度上白跑 826 个高度、并持续消耗 2 核 mongo 的算力。**
退避不减少失败次数，但能把无效消耗降到接近 0。

### 5.6 heal 不再打断 bell

healwatch 现在是 `stop lotus bell statebell tmpbell` → 写 badger → 重启。
bell 是被害者也是修复对象，这个自相矛盾要解掉。
可选方向（需实测确认锁的粒度）：只停 lotus（badger 锁在 lotus 侧）+ bell 延后写入，
或让 heal 与 bell 通过一个信号文件协调，把停 bell 的时机放在 bell 自己换批的空档。

## 六、落地次序与风险

| 阶段 | 内容 | 风险 |
|---|---|---|
| 1 | **退避 + 日志可观测**（失败时打印「已成功的 part 数 / 总 part 数 / 本批进度」） | 极低，不改提交语义 |
| 2 | **台账 + 部分提交**（5.2 + 5.3） | 中：要保证 `FinalHeight` 的连续前缀语义不被破坏；下游（聚合器 → 同步器）按高度读，退回旧区间不会出错 |
| 3 | **自动回填闭环**（5.4 + 5.6） | 中高：涉及停服务写存储，须先在 cali 验证 |

**每一步的验收判据**：不是「进程 RUNNING」，而是
`done-parts` 是否推进、`FinalHeight` 是否前移、被登记的区间是否在 `healed_at` 出现后补齐。

## 七、反面意见（记录在案，别当成已解决的共识）

部分提交会让站点出现「一段旧、其余新」的形态，这在数据产品上未必可接受——
用户可能更希望「要么全新、要么明确报错说缺一段」。这一点**尚未与业务方确认**，
在选择 5.3 之前应先问清产品期望。若产品要求「不允许任何一段旧」，
则退而采用「5.1 分级 + 5.5 退避 + 5.6 不打断」，把失败影响范围缩到「少作废几轮」而非「允许部分提交」。
