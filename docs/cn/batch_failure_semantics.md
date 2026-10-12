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

### 8.11 实现现状（2026-10-11，分支 `feat/bell-batch-failure-semantics`）

三段已全部实现在同一分支上，**均未合入 master、未部署、未启用任何 skippable 类**。

| 段 | commit | 内容 |
|---|---|---|
| 1 | `0685d783` | 配置项 `RoundBackoffInitial/Max`（5m/30m）+ `ra.go` 轮间指数退避（纯函数 `nextBackoff`，配置异常退化为「不退避」）+ `extract.go` part 失败错误带进度 |
| 2 | `694ed4fd` | `failure_class.go`（分类器，**默认不可登记**）+ `options.go` 的 `SkippableErrorClasses`（**默认空**）/`MaxUnpublishedRange`（2000）+ `partRunner`（登记 + 后续 part 只抽取不落库 + 护栏）+ `ledger.go`/`model/skipped_part.go` 台账 |
| 3 | `58dabf8e` | `prefix_commit.go`（`prefixLenBeforeFirstHole` + `commitPrefix` 单一发布咽喉）+ `Segment.Extract` 只推前缀 |

**契约变更（第 3 段）**：`ExtractTipSets` 的返回值语义变为
「**返回 nil error 不代表没有洞**」——已登记的洞通过 `failures` 返回，
由上层决定发布到前缀末。只有**致命错误**（不可登记类 / 台账写失败 / 异步落库失败）才返回非 nil error。

**已覆盖的发布路径**：

- ✅ `Segment.Extract`（daemon 主路径）——`commitPrefix` 只喂前缀给 `Aggregate` /
  `SaveFinalHeight`(或 `SaveStateFinalHeight`) / `updateBoundary`；
- ✅ `RaCailum.WalkExtract`（**离线抽取路径**，`bell offline extractor`）——
  子代理主动上报了这条路径会跨洞聚合，已修：同样只聚合前缀，洞在批首则直接拒绝。
  导出包装 `PrefixLenBeforeFirstHole` 保证两条路径复用同一份判据、不会漂移。

**尚未覆盖（有意保留 + 一处遗留）**：

- ⚠️ `Segment.ExtractToTemporaryDB`（tmp 路径）**未改**：有洞时仍整批弃。tmp 是近链头读路径，
  语义不同，改它需要单独评估。
- ⚠️ **遗留（已修，2026-10-11）**：`RaCailum.Aggregate(lo, hi)`（手动触发聚合）接受任意区间、原先不查台账 ⇒ 跨洞触发会产出跨洞派生表。已加防护（`segment.CheckNoCrossHoleAggregation` + `RegisteredHolesInRange`）：查到未修复的洞 ⇒ **拒绝并回带洞区间**（不自动截断到前缀）；**台账查询失败 ⇒ 失败关闭**。实测校准：
  - 真库上跑同形 filter（只读）⇒ 无报错、`hits=0`、集合尚未创建 ⇒ **今天不会误拦**；
  - **仓内暂无调用方**（`grep` 全仓无 `Aggregate(ctx, lo, hi)` 的调用点）⇒ 属防御性加固，不是当前线上风险，**不必为它单独换件**，随下一次换件一起上。

**发布路径全覆盖核查（改动是否真的堵住了所有口子）** —— 所有「会把数据发布出去」的路径：

| 路径 | 是否覆盖 | 手段 |
|---|---|---|
| `Segment.Extract`（daemon 主路径） | ✅ | 发布全走 `commitPrefix`，只吃洞前前缀；洞在批首不发布、不动水位 |
| `RaCailum.WalkExtract`（离线抽取） | ✅ | 复用 `PrefixLenBeforeFirstHole`；洞在批首直接拒绝 |
| `RaCailum.Aggregate`（手动触发） | ✅ | 聚合前查台账，覆盖到未修复洞即拒绝（fail-closed） |
| `Segment.ExtractToTemporaryDB`（tmp 路径） | ➖ 有意不改 | 有洞仍整批弃；tmp 是近链头读路径，语义不同，单独评估 |
| `grafana` / `persist` / 控制器里的 `*.Aggregate(...)` | ✅ 无关 | 都是 mongo 读侧聚合（取数/统计），不是发布写入 |

⇒ 判据：改完必须 `grep` 出**所有** `Aggregate(` 调用点逐个定性（发布写入 / 只读取数），只改一条会漏（本次就在离线路径上漏过一次）。

**台账**：集合 `SkippedPart`，去重靠 `_id = "<from>-<to>"`，
依赖 `lib/mgoutil/docdb.go:23` 的 `options.Update().SetUpsert(true)`（已核实），
**无需额外建索引/集合**（mongo 首次写入自动建集合）。

**启用方式**：把 `[Segment.Extract] SkippableErrorClasses` 配成 `["state"]`（先只开 state）。
**不配这个键 = 完全维持现状**——这是本改动最重要的安全属性：回退是改配置，不是回滚二进制。

### 8.12 灰度步骤与执行记录

**cali 执行记录（2026-10-11）**

| 步骤 | 时间 | 结果 |
|---|---|---|
| ① 只换二进制、**不配** `SkippableErrorClasses`（零行为变化） | 12:48 | 完成。构建用 `make build-bell-calib`（bell 必须在目标主机上编，cgo + filecoin-ffi），产物版本串 `1.37.0+calibnet+git.d37ba7bb`、不含 `+mainnet+`；**5 个共用该二进制的 supervisor 程序**（`bell`/`statebell`/`tmpbell`/`fix-actormessage`/`fix-actormessage-tmp`）全部停 → 原子 `install` → 起。复核：版本串对上、水位 4145568→4145589 在涨、错误分桶（`extract job failed`/`could not find`/`failed to persist`/`panic`）全 0、回滚锚点 `deploy-bak/bell.bak-20261011-124840` 与换装前 sha 一致 |
| ② 配 `SkippableErrorClasses = ["state"]`（**只对主 `bell`**） | 13:12 | 完成。配置备份 `deploy-bak/bell-config.bak-20261011-131248`；重启后 `bell=RUNNING`、键仍在且未被注释（⇒ bell 启动**不**重写 config，手加的键不会被洗掉）；水位 4145610 在涨、ERROR=0 |
| ③ 换到分支 tip（含 heal 闭环 + 跨洞防护） | 15:05 | 完成。`make build-bell-calib` 产物 `+git.2f808878`、不含主网标记；5 个共用实例停→原子换→起；复核 `/proc/<pid>/exe` 全指新件、回滚锚点 `deploy-bak/bell.bak-20261011-150533`（sha 与换装前一致）、配置未被改写、错误分桶全 0、`reconcile healed parts failed` = 0 |
| ④ 装上「四判据检测器」（不需要真洞就能看状态） | 15:01 | 完成。`/opt/filscan-holewatch/holewatch.sh` + 一行 crontab（每 5 分钟）；只读 bell 日志 + 聚合器水位；指标推**独立 job 组** `filscan_holewatch_cali`（避免同名族 HELP/TYPE 不一致打断整个 `/metrics` gather）。自检三档：现役日志（未撞过洞）/ **历史真实缺块日志（c3=1，证明判据在真实数据上有效）**/ 合成日志（c1=c2=c3=1）。实测 cron 已在 15:05:02 自行跑过一次 |
| ⑤ 等下一次撞洞验收四条 | 待发生 | 见下。**注意：这不阻塞任何事**——承重机制已在真库上逐条验证过（见 §8.14 与下方「真库验证」），撞洞只是最后的线上确认 |

**为什么只对主 `bell` 开**：撞洞并卡住的批次循环是主 `bell` 的（其 config `OnlyExtractState = false`，状态抽取在它的 part 里跑）。`statebell` 是另一路（`OnlyExtractState = true`、独立 repo/mongo），`tmpbell` 是近链头 tmp 路径。灰度的原则是**一次只动一条路径**，等 ③ 有结论再评估是否扩到 `statebell`。

**⚠ 前置：必须先验「分类器认不认真实故障文本」**，否则开关可能永远空转、且无人发现。
`classifyPartFailure` 是纯子串匹配 ⇒ 直接把它的规则拿去数真实日志，等价于把分类器跑在真实文本上：

| 子串（= 分类器规则） | cali（全部日志） | 主网（近 6 份日志） |
|---|---|---|
| `ipld: could not find` / `failed to load hamt node` / `failed to load state tree`（**state 类，可登记**） | **372 / 360 / 360** | 54 / 12 / 12 |
| `not minimally encoded` / `failed to unmarshal`（data 类，可登记） | 0 / 0 | 0 / 0 |
| `async persist wait timeout` / `context deadline exceeded`（传输类，**设计上不可登记**） | 0 / 0 | 11 / 74 |

cali 真实报错原文（多行，尾段）：
`… tipset execution failed: making vm: creating fvm opts: loading state tree: failed to load state tree <cid>: failed to load hamt node: blockstore get: ipld: could not find <cid>`
⇒ cali 的故障形态**正中 state 类**（三个子串全中），开关不是空转；且 cali 上 data 类与传输类各 0 ⇒「只开 state」正好够。

⇒ **主网要分清两件事**：它近期的失败里**传输类占多数**（74 次 `context deadline exceeded` + 11 次 `async persist wait timeout`，即 B 档治理的那一类），这类**故意不可登记**（瞬时故障不该固化成永久洞）；`state` 类在主网确实也发生（12 次 `failed to load hamt node`）。**所以开 `state` 对主网有用，但别指望它治主网那次落库超时事故**——那是另一条线（作业 deadline + B 档调参）治的。

**验收四条（撞洞时逐条核对，缺一条都算没生效）**：
1. `SkippedPart` 集合出现台账行；重复撞同一洞时 `attempts` 递增，而**不是**新增行；
2. `final_height` 停在洞前，**绝不越过**；
3. 日志里**同一轮出现多个不同高度**的 `extract job failed`——这是本次改动的核心行为（一轮暴露全部洞，取代「撞一个停一次」）；
4. heal 补齐后水位自然追平、台账行出现 `healed_at`。

**⚠ 坑一：启用开关的键在生成的配置模板里找不到。** `bell cfg init` 产出的模板里**没有** `SkippableErrorClasses`，因为编码器不输出值为 `nil` 的 slice（同段 `AllToCheckTableList` 有非空默认值才渲染）。**这不等于配了不生效**——实测 `[Segment.Extract] SkippableErrorClasses = ["state"]` 解码为 `[state]`，且负对照（赋成字符串）报
`incompatible types: TOML value has type string; destination has type slice`，证明键确实绑定到字段。
⇒ 该键必须**手工加在 `[Segment.Extract]` 段头之后**，别在模板里找它。

**⚠ 坑二：主机自己的仓库可能比现役二进制还旧。** cali bell 主机的 `/root/shuqi/londobell` 停在
`1af5aca3`（NV29 合并），而现役二进制是 `e419c1ad`（其下游）——**从主机 master 直接编会静默回退已上线的修复**。
⇒ 一律 `cp -a` 出副本、检出目标提交、核对 ffi gitlink（必须与树里已装的 `libfilcrypto.a` 一致，否则版本串会骗人）后再编。cali 是 x86_64。

### 8.13 三项前置求证（2026-10-11 已完成，结论如下）

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

### 8.14 heal 闭环：`healed_at` 的唯一写入路径（2026-10-11 补）

**缺口**：`model.SkippedPart.HealedAt` 此前**没有任何代码写入**（模型注释写「本段留空，heal 闭环是后续段的事」）。
后果不是「少一条验收」，而是**防护永久生效**——跨洞防护按 `healed_at: null` 判「未修复」，一旦登记过一条洞，
那条区间会永远拦住手动聚合，台账里也分不清「洞已补好」与「洞还在」。

**判据（单一、可证）**：本实例**已发布的最终水位越过洞的末端**（`to_epoch ≤ watermark`）。
理由：水位只在本轮真的抽取**并落库成功**后才前移（见 §8.3 的 `commitPrefix`），
所以「水位 ≥ 洞末端」**等价于**「这段数据已重抽且正确」——不需要额外的证明机制。

**实现要点**（`ReconcileHealedParts`）：

- 调用点：`commitPrefix` **返回 nil 之后**（只有发布落地才存在「本轮新水位」）。
  **必须每轮都试**，而不是只在「本轮有洞」时试——洞的愈合恰恰发生在**干净轮次**
  （本轮整批成功、水位一举越过此前登记的洞末端），只在有洞时试会让洞永远等不到销项。
- 每轮最多一次（`Extract` 每轮只走到那里一次），绝不下沉到 partRunner 按 part 调。
- **失败只记日志、不影响批次**：发布已完成、数据是对的，销项失败不危险，不该把已正确的一批判成失败。
  （与「台账写不进去就不允许跳过」那条铁律不同——那针对的是危险的「跳过」动作。）
- 各实例只对自己**实际抽取过**的类别背书：主 bell（`OnlyExtractState=false`）为 `state`+`data`，
  `statebell` 只为 `state`（避免用「只抽状态的水位」去证明一个 data 洞已修复）。
  `statebell` 在 `EnableExtractState` 关闭时 `commitPrefix` 不前移任何水位 ⇒ 此时跳过对账。
- **默认零影响**：`SkippableErrorClasses` 为空时台账没有行 ⇒ 无候选 ⇒ **零写入**。

**★ 实现中踩到并修掉的一个规格缺陷（值得单列）**：初版用**一条 `updateMany`**（我的规格写错了），
而 `lib/mgoutil/docdb.go` 的 `Update` 自带 `options.Update().SetUpsert(true)`，该 filter 又**没有任何等值条件**
⇒ 稳态（每轮的 filter 恰恰都无匹配）下**每轮插入一行垃圾**。真库实测：

```
updateMany(predicate, {$set:{healed_at}}, {upsert:true}) => matched=0 modified=0 upsertedId=<新 id>
集合行数 0 → 1 → 2 …          每轮 +1，约 1000 行/天
插入行字段 = ["_id","healed_at"]   ⇒ 缺 from_epoch/to_epoch/error_class
```

**修法**：先 `Find` 候选 → **无候选直接返回（零写入）** → 有候选才按 **`_id` 等值 filter** 逐个 `Update`，
并带 `$setOnInsert` 写全业务字段作双保险（万一竞态真发生插入，插进去的也是完整正确的行）。
真库复验：**无候选时集合根本不被创建**；造一条洞后按 `_id` 销项，行数仍为 1、无垃圾行；
已 healed 的不再入候选（不覆盖首次 heal 时间）；水位未越过的不入候选。

⇒ 该 API 的语义取决于 **filter 里有没有等值条件**：`recordSkippedPart` 用 `{_id:"<from>-<to>"}` 等值过滤，
它的 upsert 恰是设计意图；同一方法用在对账上就成了 bug。**回归测试要钉的是「无候选 ⇒ 一次 Update 都不发生」。**

⇒ 至此 §8.9 验收第 4 条（heal 闭环）从「无法成立」变为**可测**。

### 8.15 2026-10-12：三个新改动，与一个推翻上线计划的事实

**判决句先行**：`fe652669`（阻断面截断发布 + 异步落库按 part 归属化）**已上主网并验证发布**；
`b201e376`（ActorBalance 只写变化 + 读取侧前向填充）在 cali 双向验证完毕，**但决定不推主网**（见 8.15.1）；
`44019e3f`（actor set 缓存）在 cali 灰度。

#### 8.15.1 ★ 主网的常规 tick 根本不触发 ⇒ 满量快照只有「零时」一次

主网主 bell 的 `/root/.bell/config`：`ActorBalanceTicks = 240000`、`Interval = 120`
⇒ 常规 tick 周期 = 240000 × 120 = **2880 万 epoch（约 400 天）⇒ 永不触发**。
日志佐证：主网所有 `actor balanced extracted` 的 epoch **全部 ≡ 2160 (mod 2880)**（本地午夜零时）；
`ActorBalance` 集合里存在的 epoch 也只有 ≡2160 那一批（约 459 天、间隔 2880）。

⇒ 两条推论：

1. **满量快照每天一次**（本地午夜），不是每 12 小时一次。主网下一次 ≈ `6450214`。
2. **`b201e376` 的写入侧改动在主网上是死代码**——它只在 tick 生效，而主网 tick 不触发；
   零时那一次它按设计**仍写全量**（richlist 与批量消费者依赖全量快照）。
   ⇒ 故 `b201e376` 与读取侧**不推主网**：零收益却要担「必须与读取侧同时上、否则余额查空」的耦合。
   留在 cali 作试验床（cali 的 tick 会触发，正好持续验证这条路径），并作为将来「让零时那次也变便宜」的底座。

#### 8.15.2 主网现在真正扛住满量快照的两件事（含一处推算，未实测）

| 防护 | 依据 |
|---|---|
| 冷索引清理（21+4 个，索引 722.7→422.7GB） | 单批插入 1566ms → **300~500ms**（慢查询 6h 总量降 16 倍）；据此推算零时那次 384 万行 ≈ **19 分钟** |
| 已上主网的 `fe652669` | 万一再被阻，水位推到阻断面并发布其前的连续前缀，站点不整段停更，且下一轮自动重抽自愈 |

⚠️ 「19 分钟」是**推算不是实测**：清理后尚未经历过零时那次写入（上一次是 10-12 00:00，早于清理）。

#### 8.15.3 `b201e376` 的关键实测（cali 双向验证）

**写入侧**——用一个过去的 tick epoch `4147200` 强制触发（`bell offline extractor --local=false`，
复制 tmpbell repo 到 /tmp 绕开 repo 锁，只抽 1 个 epoch，写临时库不碰生产库）：

```
tipset/tipset.go:1375  actor balanced extracted
{"epoch":"4147200","mode":"changed","rows":34,"considered":199501,"terminated":0,"prev":"4146934"}
```

⇒ `mode=changed`；写 **34 行**而非 384 万行（**降幅约 11 万倍**，实测变化率 0.017%）；
`prev=4146934` 正是算出来的上一期零时全量（`4146934 ≡ 2614 (mod 2880)`，cali 的零时残差是 2614，主网是 2160）。

**两条否证**（省掉后来者弯路）：tmp 路径永远验不到 ActorBalance
（`extractActorBalance` 首句就 `if tmp { return nil }`）；`statebell` 也不写它
（其配置 `EnableExtractActorBalance=false` 且 `ActorBalanceTicks=240000`）。

**读取侧**——cali `londobell-api-aggregators` 换到 `b201e376`：

| 请求 epoch | 换前 | 换后 |
|---|---|---|
| `4147200`（快照期） | 有数据 | **逐字节相同**（无回归） |
| `4148003` / `4147800` / `4148150`（非快照期） | `data:null` | 返回最近一期余额（前向填充生效） |
| `5000000`（超链头） | — | `null`（越界保护） |

richlist：`start=4147200` 与 `start=4148150` 返回字节数相同（70873）⇒ 仍取最近全量快照，语义未变。

#### 8.15.4 主网前向填充查询的执行计划（只读实测）

查询形状（`balance_forward_fill.go` 原文）：`$match{Epoch:{$gte:StartEpoch,$lt:EndEpoch}, Addr}` → `$sort{Epoch:-1}` → `$limit 1`。

```
计划：LIMIT <- FETCH <- IXSCAN[Epoch_1_Addr_1] direction=backward，无 COLLSCAN、无阻塞 SORT
indexBounds 把 Addr 当等值推进 Epoch 区间内 ⇒ 是 seek，不是范围扫
三个用例（最近在变 / 最远回看 / 不存在的地址）：0~4ms，keysExamined 1~2，docsExamined 0~1
```

集合现状：17.06 亿文档 / 589.5GB / 索引 193GB（`_id_` 118.9GB、`Epoch_1_Addr_1` 29.95GB、`Epoch_1_Balance_-1_Code_1` 44.2GB）。

★ **唯一风险边界**：`Epoch_1_Addr_1` 是 **sparse** 索引 ⇒ **任何丢掉 `Epoch` 条件的 `ActorBalance` 查询会退化成 589GB 全扫**（已实测）。
读取侧永远带 `Epoch` 区间 ⇒ 安全；**将来新增查询必须带 `Epoch` 条件**。
代价是 O(窗口内不同 epoch 数) 次索引 seek（约 1ms/个）；今天窗口内只有 1 个 epoch，最坏 1~2 个。
`{Addr:1,Epoch:-1}` 理论更优（不存在的地址也只需一次 seek），但收益为零 ⇒ **不建**（另建约 30~45GB、多小时）。

#### 8.15.5 `44019e3f`：actor set 不再每轮重建

实测依据与修法见技能 `filscan-data-pipeline-ops/references/bell-actor-set-rebuild-cost.md`。
**关键否定**：持续刷的 `code not found ... exists in init state` WARN **几乎不花钱**（else 分支只打一行日志），
它不是吞吐瓶颈；真正贵的是**每轮重建**（单次 3分13秒~8分00秒，占一轮 1/3~1/2，而 actor 数 25 分钟只涨 7 个）。
修法复用仓内已有缓存机器（网络版本作 key + 30 轮/2000 epoch 有界兜底），未动数据语义；
连带修掉 `loadActorSet` 返回缓存原指针、调用方锁外读可变字段的 data race。

#### 8.15.6 教训：同一个坑两次都踩在「没量清楚就归因」

1. 把「删除冷索引」说成放行了卡住的批次——实际发布**早于**删索引 2 分钟（时间戳一查就穿）。
2. 把「聚合阶段静默」当成卡死——同一阶段历史耗时 24 秒到 6 分 46 秒都有（聚合器只在跑完打一行）。

**规矩**：先拿时间戳与历史同类耗时对齐再下结论；**日志静默不等于卡死**；删除/变更与现象同时出现时，
先排时间先后再谈因果。

#### 8.15.7 `44019e3f` 的 cali 灰度实测（10-12 10:40）与一处量级更正

**机制证明（两个非 tmp 进程都生效）**：

```
换前基线（近 8MB 窗口）：23 轮 = 23 次 actor set load（1:1，每轮重建），actor set reused = 0
换后 bell（非 tmp）：    4 轮 ⇒ load=1（首轮冷构建）+ reused=3
换后 statebell（非 tmp）：同样 ⇒ load=1 + reused=3
对照组 tmpbell（原有缓存）：76 轮 ⇒ load=4、reused=72、
  actor set cache invalidated, rebuilding {reason: budget-exhausted} 2 次，间隔 ≈31 轮 ⇒ 与 budget=30 相符
```

复核：5/5 程序 RUNNING、每个 pid exe=`/root/bell`、config sha 不变、panic/ERROR/failed-to-persist 全 0、水位 4148108→4148193。

⚠️ **量级更正（重要）**：cali 的 actor set 只有 **47.5 万条**（`count` 199504 + `init-state` 275481），
单次重建实测 **p50=1.14s / p90=1.81s / max=26.31s**（128 轮样本）——**不是 §8.15.5 里的 3分13秒~8分00秒**，
后者是**主网**（384 万 + 386 万条）的量级。⇒ 该改动的收益**随规模放大**：cali 上只有 ~1.1s/轮（看不出），
**主网上才是它要解决的问题**（每轮 3~8 分钟）。

#### 8.15.8 上线裁定（10-12 10:4x）

| commit | 裁定 | 依据 |
|---|---|---|
| `fe652669` | 已在主网 | 已实测发布 |
| `b201e376`（改动 4） | **不推主网** | §8.15.1：主网 tick 不触发 ⇒ 写入侧在主网是死代码，零收益却要担读写耦合 |
| `44019e3f`（改动 5） | **已推主网**（10-12 10:45:45 换件） | 纯性能、不动数据语义、无读写耦合，且收益恰在主网量级 |

**改动 5 主网换件实测（10-12 11:19 独立复核）**：

```
version 1.37.0+mainnet+git.44019e3f.dirty · sha aa0feb0f6abbc5d79d958127b3a992dfaa868de7a62bbd44297d99669e6a86a4
config sha 837813b210d184a2e67602834d5d5af8122ecb271a909375298dd2f68e088d89（**逐字节未变**）
8 个程序全 RUNNING · pgrep 到的 exe 全指 /root/bell · 零 panic/ERROR/failed-to-persist/阻断面
换件后 4 轮：actor set reused=3、actor set load=1（首轮冷构建）
换前每轮 actor set load→loaded 实测 98.9~193.6 秒（5 轮样本）；换后同量级轮次 1 分 0 秒
⇒ 每轮省约 1.6~2.5 分钟，占典型轮耗时（2m42s~4m0s）的 40~50%
```

（注：§8.15.5 与技能笔记里的「3 分 13 秒~8 分 00 秒」是另一个窗口的量级；实测区间是 **1.6~8 分钟**，取数口径一致时每轮省 1.6~2.5 分钟。）

★ **换件后暴露的下一个吞吐嫌疑**：某轮 8 分 10 秒的长尾**不是** actor set 造成的——
该轮分片 11:07:49 已全 done，但 `part [6448913,6448920]` 到 **11:15:16** 才落库，
期间出现 `ActorAddress 18/3844882`、`FilSupply 1/1`（该 part `insertOps=7653`）。
「3844882」若是每轮全量重算地址映射的 considered 数，则属与 actor set 同一类的浪费（调查中）。

**改动 5 的风险边界**：缓存 key = 网络版本；另有 30 轮 / 2000 epoch 有界兜底；
集合里查不到的 actor 走 per-message 兜底 `stm.LoadActor` 并回写；
「actor 存在而 code 变了」不产生 MISS，由两个有界窗口封顶（真实 code 变更来源是升级迁移 ⇒ 被版本 key 抓住）。

⚠️ **附带脚枪（必须记）**：`44019e3f` 是 `b201e376` 的后代，即**主网新件里也含改动 4 的代码**。
在主网当前配置（`ActorBalanceTicks=240000`，tick 永不触发）下它是**惰性无害**的；
**但若将来把 `ActorBalanceTicks` 调小，必须先上读取侧（前向填充）**，否则余额查询会出现空白。
