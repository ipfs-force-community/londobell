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
| 数据级 | 该 part 内容有问题（如 exec-trace 解不出、毒高度） | 登记 → 后续 part 照常抽 → **发布仍停在洞前** → 事后回填 |
| 状态级（不可恢复） | 节点侧状态已裁掉，再等也不会好 | 登记 → 发布停在洞前 → 借归档源回填或长期留账 |
| 传输级 | 网络/超时/连接被打断 | **不跳过**，保持无限重试（现状） |

注意「登记 → 跳过」的准确含义：**跳过的是「让这个坏 part 继续拖着整批失败」，不是「把它发布出去」**。
被登记的洞之后，`FinalHeight` 依然停在该洞之前（见 5.3），所以登记 + 继续抽的目的只有一个——
让已成功的部分有机会先提交、并让 heal 有机会在下一轮换批前补齐。

关键分界：**同一错误连续出现 N 次且高度/内容固定 ⇒ 数据级（登记有意义）；每次失败的位置都在变 ⇒ 传输/资源级（重试有意义）。**

### 5.2 台账：一张表，语义照抄上游

新增 `chain.sync_skipped_parts`（或等价命名，bridge 到既有命名风格）：

- `(skipped_from, skipped_to)` 区间、`error_class`、`cid`（触发失败的那个根 CID）、
  `attempts`、`first_seen`、`last_seen`、`healed_at`（回填成功后回填该列）；
- **唯一键 `(skipped_from, skipped_to)`** ⇒ 同一区间重复失败只更新 `last_seen`/`attempts`，不插新行；
- **写不进台账就不允许跳过** —— 照上游那条铁律，否则会变成静默跳洞。

### 5.3 提交粒度：只推到「已成功部分的连续最前头」，绝不跨洞发布

这是本设计里**唯一被正确性约束锁死**的选择，不是可以权衡的产品偏好。

`FinalHeight` 当前只在整批成功后写。改为**按已成功的连续前缀推进**：
`SaveFinalHeight` 的入参从「整批」变成「本批已成功的最大连续 part 上界」——**遇到洞就停在那里**，洞之后即使已经抽好的 part 也不发布。

为什么必须这样：很多派生表（曲线、N 日增幅、累计奖励、榜单对比）是**跨高度算出来**的。
一旦发布跨过一个洞，这些表会「接口 200、日志无错、数值错误」——把洞两边的差当成真实变化。
这是最难发现的故障形态，**所以宁可少恢复，也绝不能跨洞发布**。

| 方案 | 水位怎么推 | 正确性 | cali 昨晚的实际收益 |
|---|---|---|---|
| **安全版（本设计采用）** | 只推到连续前缀，遇洞停住 | ✅ 洞永远不会被跨过发布，派生表不可能被污染 | **原写「884/3487 高度」有误，已更正——见 §8.0：单靠前缀提交收益≈0，真正的杠杆是「登记后继续往后抽」** |
| ~~激进版~~ | 跨过洞继续往后发布 | ❌ 洞两侧派生值错误，须等回填后**重算**才恢复 | 31.5 小时全恢复 |

激进版要成立，前提是先证明「回填 → 重算所有跨洞派生表」这条闭环能自动跑通。
**该前提证伪之前不允许启用**，本文不把它列为可选项。

⇒ 本设计的收益口径要说清楚：**不是「31.5 小时 → 0」，而是「31.5 小时 → 约 7 小时」**，
剩下那一截仍然等 heal 补上块之后自然追平——那一段等的是同一个 heal，只是不再拖着已成功的部分一起作废。

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

## 七、边界与未解决项

1. **本设计不允许跨洞发布**（见 5.3），这是正确性约束。因此它的收益有上限：
   cali 那类事故从「31.5 小时」缩到「约 7 小时」，**不是恢复到 0**。
   要真正回到 0，只能靠 heal 把块补回来然后自然追平——本设计只是让它别再拖着已成功的部分一起作废。
2. **「回填后重算跨洞派生表」尚未实现，也没验证过**。在这条闭环被证明之前，
   任何形式的跨洞发布都不允许启用（否则就是 200 + 无错日志 + 错数值）。
3. **台账的写入者与磁盘空间**：坏 part 持续发生时，`sync_skipped_parts` 会增长；
   长时间不愈合的区间要不要告警、要不要升级为人工介入，需要规则。
4. **calibnet 与 mainnet 的构建标签不同**（`--tags=calibnet` / 主网标签），
   改动必须两网分别构建、分别发布，且**不能在另一网正追赶时换件**。

---

## 八、实现规格（可照着写代码）

> 行号对应 2026-10-11 的仓内 HEAD（`londobell` master）。

### 8.0 先更正一条收益口径（§5.3 的表）

§5.3 写「安全版可恢复 884/3487 高度」，**这个数字的推导有误，须按机制重算**。

实测 cali 那批：区间 `[4140557, 4144044]`、`TipSetPartSizeLimit = 16`（`done-parts: 1 → done-tss: 16`），
**首个洞 4140561 落在第 1 个 part 内（批内第 5 个高度）**。

⇒ 若按 **part 粒度**取「连续前缀」，前缀 = **0 个 part**，一个高度都恢复不了；
若按 **tipset 粒度**，前缀 = 4 个高度，也约等于 0。
**「只推连续前缀」这一条本身几乎不产生收益。**

真正的杠杆在另一半——**「登记坏 part 后继续往后抽」**：

| 机制 | 作用 |
|---|---|
| 只想推前缀、遇洞即停 | 收益 ≈ 0（洞在批首时前缀为空） |
| **继续往后抽并逐个登记** | bell 在**同一轮**内撞出**所有**洞 ⇒ heal 一次拿到全部 CID、一轮修完。**注意：只抽取、不落库**——见 §8.3 的实测修正 |

原流程是「撞一个洞 → 整批弃 → heal 修 1 个 → 再撞下一个」，6 个洞 ⇒ 6 轮 × (1~3 小时)；
改成继续往后抽以后，**6 个洞在同一轮内全部暴露**，heal 一轮修完。
⇒ 这才是 31.5 小时 → 小时级的来源。**两条必须一起做，单独做「前缀提交」没有意义。**

### 8.1 控制流事实（已逐行核对）

`Segment.Extract`（`segment.go:231–314`）：

```
lo = bound.Lo;  hi = bound.Hi
tipsets = ExtractLinkedTipSets(cs, rawts, &hi)   // 区间 (hi, rawts]
tipsets = tipsets[1:]                            // 去掉 hi 自身
按 Confidence 裁掉尾部未确认的
ExtractTipSets(ctx, tipsets, false)   ← ① 批抽取，这里失败 = 整批弃
Aggregate(ctx, tipsets)               ← ② 跨高度聚合（必须只喂前缀）
SaveFinalHeight(tipsets[len-1])       ← ③ 对外发布水位
updateBoundary(tipsets[len-1], nil)   ← ④ 段边界前移
```

`ExtractTipSets`（`extract.go:193–295`）：

```
for tsDone < size {
    part := tss[tsDone : tsDone+TipSetPartSizeLimit]
    if err := s.extractPart(pctx, part, tmp); err != nil { return err }   ← 注入口 ①
    tsDone += len(part); partDone++
}
if err := waitAsyncPersist(&pctx.asyncPersistWaitGroup, WaitTimeout); err != nil {  ← 注入口 ②
    return fmt.Errorf("error occurs in async persist: %s", err)
}
```

`extractPart`（`extract.go:297–428`）：part 内每个 tipset 一个 goroutine，结果落到 `docs[ti]`；
`ewg.Wait()` 返回**首个**错误；随后 `insertManyWithTimeout(docs)`（`Persist.Async=true` 时进 `asyncPersistWaitGroup`）。

`Aggregate`（`aggregate/aggregate.go:86–141`）：**逐 tipset 独立**跑各 source 的 mongo pipeline
（`for ti := range tss`）⇒ **传入前缀切片是安全的**，不会跨洞计算。

### 8.2 改动一：失败分类 + 登记 + 继续（`extract.go`）

**注入口 ①** 由「return err」改为「分类 → 可跳过则登记并继续」：

```go
var failures []PartFailure
for tsDone < size {
    part := tss[tsDone : tsDone+s.opts.Extract.TipSetPartSizeLimit]
    if err := s.extractPart(pctx, part, tmp); err != nil {
        if pf, ok := classifyPartFailure(part, err); ok {
            failures = append(failures, pf)   // 登记，不中断
            // 注意：continue 而不是 return
        } else {
            return err                        // 传输级：保持现状，整批弃
        }
    } else { ... }
    tsDone += len(part); partDone++
}
```

**注入口 ②** 目前 `waitAsyncPersist` 只返回一个合并错误，**无法归因到哪个 part**。
必须改为按 part 收集：把 `asyncPersistWaitGroup.Go(...)` 的闭包捕获该 part 的序号与区间，
失败时写入一个 `chan partPersistErr`；`waitAsyncPersist` 改为「等全部 goroutine 结束后读该 chan」。
否则「落库超时的 part」无法登记，而这正是主网那类故障的形态。

**分类器**（新增，建议放 `racailum/segment/failure_class.go`）：

| 判据（按 `err.Error()` 子串匹配） | 类别 | 处置 |
|---|---|---|
| `ipld: could not find`、`failed to load hamt node`、`failed to load state tree` | `state` | 登记 + 继续 |
| `not minimally encoded`、`failed to unmarshal`、业务码非 0 | `data` | 登记 + 继续 |
| `extract job timeout`（`*extractJobTimeoutErr`） | `transport` | 不登记，整批弃 |
| `async persist wait timeout`、`context deadline exceeded`、`connection ... incomplete read` | `transport` | 不登记，整批弃 |
| 其余（未知） | `unknown` | **默认整批弃**（保守：宁可停，不可静默跳） |

**要点**：默认分支必须是「弃」不是「跳」。分类器的白名单越窄越好。
`*extractJobTimeoutErr` 用 `errors.As` 判类型，不要靠字符串。

### 8.3 改动二：提交只推「连续前缀」（`segment.go`）

`ExtractTipSets` 的返回值从 `error` 改为 `(PartFailures, error)`（或经 `Segment` 上的字段回传）。

```go
// segment.go Extract() 内，拿到 fails 之后：
prefixLen := computePrefix(tipsets, fails)   // 连续成功的高度数
if prefixLen == 0 {
    return fmt.Errorf("batch head blocked by hole(s): %v", fails)  // 保持现状：不动水位
}

prefix := tipsets[:prefixLen]
if err := s.Aggregate(ctx, prefix); err != nil { return err }   // ★ 只喂前缀
... SaveFinalHeight(ctx, prefix[prefixLen-1])                   // ★ 只推前缀末
if err := s.updateBoundary(ctx, prefix[prefixLen-1], nil); err != nil { return err }
// 坏 part 已登记；下一轮从这里继续，撞同一洞 → 命中台账 → 不重复登记
```

**★ 两处是正确性红线**：`Aggregate` 与 `SaveFinalHeight` 都必须只吃前缀。
漏掉 `Aggregate` 这一处，就等于**跨洞发布派生表**（200 + 无错日志 + 错数值）。

**注（2026-10-11 实测已推翻原写法，务必按新规则实现）**：

原写「继续往后抽的那些 part 对外不可见（聚合器只读 ≤ `final_height`）」——
**实测这是错的**。在 cali 上取 `final_height = 4145316`，再查
`POST /aggregators/traces {"start":4145317,"end":4145327}`（**水位之上**），
**返回了 `Epoch:4145317` 的完整行**。⇒ **聚合器不按水位拦截区间查询**，
凡写进 mongo 的数据，客户端用显式区间就能读到。

这不是异常：`FinalHeight` 本来只在整批成功后才前移，而各 part 是**边抽边落库**的，
所以「已写入但未发布」在**正常运行时每一批都存在**。区别在于——
**今天的可见区间永远是连续的**（失败即停，没有洞）；一旦改成「登记后继续往后抽并落库」，
可见区间就会变成 `[.., 洞) ∪ (洞, ..]`，**客户端跨洞查询时拿到两侧数据、无从察觉中间缺失**
——这正是本设计要避免的形态。

⇒ **规则修正：坏 part 之后的 part 只做「抽取以暴露错误」，不得落库。**

```go
// extractPart 增加 persist 开关：
//   persist=false ⇒ 照常抽取（该失败的照样报错、被登记），但 docs 丢弃、不写 mongo
for tsDone < size {
    part := tss[tsDone : ...]
    err := s.extractPart(pctx, part, tmp, /*persist=*/ len(failures) == 0)
    if err != nil {
        if pf, ok := classifyPartFailure(part, err); ok {
            failures = append(failures, pf)   // 之后所有 part 都以 persist=false 跑
        } else { return err }
    }
    tsDone += len(part)
}
```

这样既拿到「一轮暴露全部洞（heal 一次修完）」这个唯一有效的杠杆，
又保证**可见区间始终连续**、不会把洞暴露给客户端。
代价：洞之后的抽取结果会被丢弃、下一轮重抽——**这是可接受的成本**（抽取不是瓶颈，无效重抽才是）。

**另需注意**：`Aggregate` 与 `SaveFinalHeight` 仍必须只吃前缀（见上），两处红线不变。

### 8.4 改动三：登记台账（新增集合）

写到 bell 的写库（与 `FinalHeight` 同库），命名随本仓风格：

```
集合 SkippedPart {
  from_epoch  int64     // part 起始高度
  to_epoch    int64     // part 结束高度
  error_class string    // data | state
  err_digest  string    // 错误摘要（截断，勿存超长）
  cid         string    // 从错误文本提取的根 CID（提不到则空）
  attempts    int
  first_seen  time
  last_seen   time
  healed_at   time|null
}
唯一索引 (from_epoch, to_epoch)
```

**铁律（照抄上游）**：**台账写不进去就不允许跳过**——写失败则按原逻辑整批弃，
否则就变成静默跳洞（用户明确否掉过的形态）。
重复命中同一区间时只更新 `last_seen` / `attempts`，不插新行。

### 8.5 改动四：护栏（防洞不愈合时无限前进）

继续往后抽有个必然副作用：**洞不愈合时，bell 会一路抽到链头**，
未发布区间无限增长、mongo 里堆积不可见的文档。

护栏：新增 `MaxUnpublishedRange`（高度）。当 `链头 − final_height > 阈值` 时
**停止继续往后抽**、只登记并告警。建议初始值取 **2000 高度**（约 17 小时）。

### 8.6 改动五：轮间退避（`ra.go` Run）

现状（`ra.go:220`）：失败后**不等待**，下个 tipset（约 30 秒）到来即重跑整段，
每轮都在已知会失败的高度上白跑。

改为指数退避：`5min → 10min → 20min`，上限 30min；成功一轮后立即复位。
接入点：`Run` 循环里 `r.Extract` 返回 error 的分支。
**注意**：退避期间 `AlertOutdatedFinalHeight`（`ra.go:233`）仍在跑，告警不会哑掉。

### 8.7 配置项（`options.go`）

```go
type extractOptions struct {
    ... 既有 ...
    // 失败分级：可登记并继续的错误类别（默认仅 state/data）
    SkippableErrorClasses []string      // 默认 {"state","data"}
    // 护栏：未发布区间上限，超过即停手告警
    MaxUnpublishedRange   abi.ChainEpoch // 默认 2000
    // 轮间退避
    RoundBackoffInitial   time.Duration  // 默认 5m
    RoundBackoffMax       time.Duration  // 默认 30m
}
```
**默认值必须与现状等价或更保守**：`SkippableErrorClasses` 上线初期建议先配成**空**
（= 完全维持现状），确认链路无误后再开——这样灰度回退是改配置而不是回滚二进制。

### 8.8 测试计划

| 层 | 用例 |
|---|---|
| 单元 | `classifyPartFailure`：上表每一行一个正例 + 一个「未知错误必须归为弃」的反例 |
| 单元 | `computePrefix`：失败在第 0 / 中间 / 最后一个 part；多个失败；无失败 |
| 单元 | 台账重复命中：同区间两次只更新 `last_seen`，不新增行 |
| 单元 | 护栏：未发布区间超阈值时停止继续抽 |
| 集成（假 dal，不连链） | 让第 k 个 part 固定失败 ⇒ 断言：台账有行；`Aggregate` 收到的切片**等于前缀**；`SaveFinalHeight`/`updateBoundary` 收到前缀末 |
| 回归 | 既有 `extract_timeout_test.go` 必须继续通过：**超时仍属 transport、仍整批弃** |
| 回归 | `WaitTimeout` 超时仍整批弃（不因新逻辑被误判为可跳过） |

### 8.9 验收判据（上线后看这几个，不看「进程 RUNNING」）

1. **台账出现行**：撞洞后 `SkippedPart` 有记录，且 `attempts` 随轮次增长而非新增行；
2. **水位停而不退**：`final_height` 停在洞前（**绝不越过洞**）；
3. **同一轮暴露多个洞**（本次改动的核心行为）：日志中出现**多个不同高度**的
   `extract job failed`，而不是每轮只报一个；
4. **补齐后自然追平**：heal 修完 → 下一轮 `final_height` 前移、台账行出现 `healed_at`；
5. 单测与集成测试全绿。

### 8.10 上线顺序

1. **只上「退避 + 可观测日志」**（§8.6），`SkippableErrorClasses = {}`——零语义变化，先验证观测面；
2. 开 `state` 类，**cali 先行**，观察一个完整的洞周期（撞洞 → 登记 → 继续 → heal → 追平）；
3. cali 稳定后再开 `data` 类；
4. **主网最后上**，且必须避开追赶期。

### 8.11 三项前置求证（2026-10-11 已完成，结论如下）

**① 段边界 `hi-epoch` 与发布水位 `final_height` 的关系 —— 稳态下完全相等。**

实测（同一时刻）：`bell segment show --name bell` 报 `hi-epoch = 4145316`，
聚合器 API 报 `final_height = 4145316`，**两者相等**。

⇒ 二者随每一批成功同步前移，**批区间恒为 `(final_height, 链头]`**。
⇒ 事故时那 1004 的差值（水位 4138552 / 批起点 4140557）是**当时恢复过程造成的异常态，不是稳态属性**
（代码顺序是 `SaveFinalHeight` 在前、`updateBoundary` 在后，后者失败不会前移，所以正常路径下边界不可能领先水位）。
⇒ **收益口径成立且更悲观**：失败一轮后水位不动、下轮从水位+1 重抽，
即**批首就是洞附近 ⇒ 「只推连续前缀」的收益确为 ≈ 0**（与 §8.0 一致）。

**② 聚合器是否按水位拦截区间查询 —— 不拦截。§8.3 原文的前提被推翻。**

实测：`final_height = 4145316` 时查 `traces {"start":4145317,"end":4145327}`
**返回 `Epoch:4145317` 的完整行**；对照水位之下的 `[4145306, 4145316]` 同样返回行。

⇒ **凡写进 mongo 的数据，客户端用显式区间都能读到**，水位只用于
「API 上报的 FinalHeight」与「计数累加状态（`GetEndEpoch`）」，**不是查询的硬过滤**。
⇒ 已在 §8.3 改为「坏 part 之后的 part 只抽取、**不落库**」。

**③ `insertMany` 的幂等保护覆盖范围 —— 只覆盖 update 路径，insert 路径没有。**

`persist.go` 有两条路径：

| 路径 | 位置 | epoch 守卫 |
|---|---|---|
| `update` | `persist.go:107–147` | **有**：`epochValue >= existingEpoch` 才写，否则 `continue` |
| `insert` | `persist.go:54`（`s.db.Insert(ctx, col, docs)`） | **无** |

⇒ 重抽并重写已存在的 insert 型文档会**撞重复键**（不是被守卫挡下），
产生 `ninserted:0` + 大体积 writeErrors 响应。
**这正好解释了实测到的现象**：主网 `bell.ActorAddress` 当日 60,607 条慢查询里
**59,985 条 `ninserted:0`**、每条约 433 ms、返回 820 KB 重复键错误。

两个直接结论：

1. **重抽不是「静默幂等」，是「报错空转」**——写不掉、白花 mongo 时间，但**不会写坏数据**
   （不会被旧 epoch 覆盖）。⇒ 它对**正确性无害**，对**性能有害**。
2. ⇒ §8.2 的对象「继续往后抽」的代价应记为**重复键空转开销**，
   这进一步支持 §8.3 的「不落库」规则：不落库就没有这批空转。
