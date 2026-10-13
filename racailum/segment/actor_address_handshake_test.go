package segment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

// 改动 6 补丁：ActorAddress 的落库交握。
//
// 语义：extractActorAddress 把本 part 发射的 ActorID 记进该 part 的 *pending* 桶；
// 该 part 落库**成功**才 Commit（pending 升格为已写、下一轮不再重发）；落库**失败/超时**
// 或未落库就丢弃 ⇒ Discard（pending 丢弃、下一轮重发）。这就堵掉了「异步落库失败时
// ActorAddress 行被静默跳过、直到进程重启才补」的空洞。

// aaIDs 构造 []abi.ActorID（本包测试用，对应 extract 包测试里的 idsOf）。
func aaIDs(vs ...uint64) []abi.ActorID {
	out := make([]abi.ActorID, 0, len(vs))
	for _, v := range vs {
		out = append(out, abi.ActorID(v))
	}

	return out
}

func newHandshakeSegment() *Segment {
	return &Segment{opts: DefaultOptions(), actorAddressState: extract.NewActorAddressWriteState()}
}

func newHandshakeCtx() *persistCtx {
	return &persistCtx{
		ctx:        context.Background(),
		log:        zap.NewNop().Sugar(),
		persistSem: make(chan struct{}, 2),
	}
}

// 成功 ⇒ Commit（pending 升格，下一轮不再重发）；失败 ⇒ Discard（下一轮重发）。
// 这是本次改动唯一要堵的正确性缺口，故正反两面都在这里锁住。
func TestPersistPartDocs_ActorAddressHandshakeCommitAndDiscard(t *testing.T) {
	fake := newFakeDocDB()
	s := newHandshakeSegment()
	s.db = fake

	pctx := newHandshakeCtx()
	part := mkTSS(t, 10, 11)
	docs := [][]common.Document{{&fakeDoc{}}}

	// --- 成功：part 落库 ⇒ Commit ---
	ok := s.actorAddressState.BeginPart()
	if got, _ := ok.Filter(aaIDs(1, 2, 3)); len(got) != 3 {
		t.Fatalf("首个 part 应发射 3 条，实际 %v", got)
	}
	if err := s.persistPartDocs(pctx, pctx.log, part, docs, true, ok); err != nil {
		t.Fatalf("persistPartDocs 不应报错: %v", err)
	}
	if fails := pctx.asyncPersist.wait(time.Second); len(fails) != 0 {
		t.Fatalf("成功的 part 不应有失败归属: %+v", fails)
	}
	if got := s.actorAddressState.Seen(); got != 3 {
		t.Fatalf("落库成功后 Seen() = %d, want 3（pending 应升格为已写）", got)
	}
	if got := s.actorAddressState.Pending(); got != 0 {
		t.Fatalf("落库成功后 Pending() = %d, want 0", got)
	}

	// 已落地的 id 不再重发。
	dup := s.actorAddressState.BeginPart()
	if got, _ := dup.Filter(aaIDs(1, 2, 3)); len(got) != 0 {
		t.Fatalf("已写的 id 不应再发射，实际 %v", got)
	}
	dup.Discard()

	// --- 失败：part 落库报错 ⇒ Discard，且下一轮重发 ---
	fake.insertErr = errors.New("insert extracted documents from tipsets: context deadline exceeded")

	bad := s.actorAddressState.BeginPart()
	if got, _ := bad.Filter(aaIDs(4, 5, 6)); len(got) != 3 {
		t.Fatalf("新 id 应被发射，实际 %v", got)
	}
	if err := s.persistPartDocs(pctx, pctx.log, part, docs, true, bad); err != nil {
		t.Fatalf("persistPartDocs 异步路径本身不应直接报错: %v", err)
	}
	fails := pctx.asyncPersist.wait(time.Second)
	if len(fails) != 1 {
		t.Fatalf("落库失败应恰好归属 1 个 part，实际 %d（%+v）", len(fails), fails)
	}
	if got := s.actorAddressState.Seen(); got != 3 {
		t.Fatalf("落库失败不得计入已写：Seen() = %d, want 3", got)
	}
	if got := s.actorAddressState.Pending(); got != 0 {
		t.Fatalf("落库失败应丢弃 pending：Pending() = %d, want 0", got)
	}

	// 下一轮：失败的 id 重新发射（这正是被堵掉的空洞）。
	again := s.actorAddressState.BeginPart()
	if got, _ := again.Filter(aaIDs(4, 5, 6)); len(got) != 3 {
		t.Fatalf("落库失败的 id 下一轮必须重发，实际发射 %v", got)
	}
	again.Commit()
	if got := s.actorAddressState.Seen(); got != 6 {
		t.Fatalf("重发并成功后 Seen() = %d, want 6", got)
	}
}

// persist=false 路径不得记账：不再调 insertManyWithTimeout，也不动 pending/seen。
func TestPersistPartDocs_ActorAddressPersistFalseDoesNotAccount(t *testing.T) {
	fake := newFakeDocDB()
	s := newHandshakeSegment()
	s.db = fake

	pctx := newHandshakeCtx()
	part := mkTSS(t, 20, 21)
	docs := [][]common.Document{{&fakeDoc{}}}

	aa := s.actorAddressState.BeginPart()
	if got, _ := aa.Filter(aaIDs(1, 2, 3)); len(got) != 3 {
		t.Fatalf("应发射 3 条，实际 %v", got)
	}

	if err := s.persistPartDocs(pctx, pctx.log, part, docs, false, nil); err != nil {
		t.Fatalf("persist=false 不应报错: %v", err)
	}
	if fake.insertCalls != 0 {
		t.Fatalf("persist=false 不得落库，实际 insert=%d", fake.insertCalls)
	}
	// persist=false 时上层根本不会传 aaPart（传 nil），也不会 Commit/Discard：
	// 发射的 id 仍留在 pending，但这条路径用的是抛掷态（见 extractPart），
	// 真实 state 因此完全不受影响——这里断言真实 state 的 seen 未动。
	if got := s.actorAddressState.Seen(); got != 0 {
		t.Fatalf("persist=false 不得计入已写：Seen() = %d, want 0", got)
	}
}

// 同一 part 内多 tipset 并发发射，落库成功后恰好一次入账（-race 下无数据竞争）。
func TestPersistPartDocs_ActorAddressConcurrentPartCommitsExactlyOnce(t *testing.T) {
	fake := newFakeDocDB()
	s := newHandshakeSegment()
	s.db = fake

	pctx := newHandshakeCtx()
	part := mkTSS(t, 30, 31)
	docs := [][]common.Document{{&fakeDoc{}}}

	aa := s.actorAddressState.BeginPart()
	batch := aaIDs(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	const workers = 8
	var (
		mu    sync.Mutex
		perID = map[abi.ActorID]int{}
		wg    sync.WaitGroup
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			emitted, _ := aa.Filter(batch)
			mu.Lock()
			defer mu.Unlock()
			for _, id := range emitted {
				perID[id]++
			}
		}()
	}
	wg.Wait()

	if len(perID) != len(batch) {
		t.Fatalf("并发 part 内 distinct id = %d, want %d", len(perID), len(batch))
	}
	for id, n := range perID {
		if n != 1 {
			t.Fatalf("id %d 被发射 %d 次，want 恰好 1 次", id, n)
		}
	}

	if err := s.persistPartDocs(pctx, pctx.log, part, docs, true, aa); err != nil {
		t.Fatalf("persistPartDocs 不应报错: %v", err)
	}
	if fails := pctx.asyncPersist.wait(time.Second); len(fails) != 0 {
		t.Fatalf("不应有失败归属: %+v", fails)
	}
	if got := s.actorAddressState.Seen(); got != len(batch) {
		t.Fatalf("Seen() = %d, want %d", got, len(batch))
	}
	if got := s.actorAddressState.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want 0", got)
	}
}
