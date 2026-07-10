package runs

import (
	"testing"
	"time"
)

// readNChunks reads exactly n chunks from the channel within the given timeout.
// It fails the test if the channel closes early or a timeout occurs.
func readNChunks(t *testing.T, ch <-chan OutputChunk, n int, timeout time.Duration) []OutputChunk {
	t.Helper()
	chunks := make([]OutputChunk, 0, n)
	for i := 0; i < n; i++ {
		select {
		case chunk, ok := <-ch:
			if !ok {
				t.Fatalf("channel closed unexpectedly after %d/%d chunks", i, n)
			}
			chunks = append(chunks, chunk)
		case <-time.After(timeout):
			t.Fatalf("timeout waiting for chunk %d/%d", i, n)
		}
	}
	return chunks
}

// readAllChunks reads all available chunks from the channel within the given
// timeout. It returns what it got when the channel is closed or the timer fires.
func readAllChunks(t *testing.T, ch <-chan OutputChunk, timeout time.Duration) []OutputChunk {
	t.Helper()
	var chunks []OutputChunk
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return chunks
			}
			chunks = append(chunks, chunk)
		case <-timer.C:
			return chunks
		}
	}
}

// ---------------------------------------------------------------------------
// Lifecycle: StartRun → Subscribe → Emit → FinishRun
// ---------------------------------------------------------------------------

func TestStartRunSubscribeEmitFinishRun(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-run-lifecycle"

	// Start the run and verify state.
	b.StartRun(runID)
	if !b.IsRunning(runID) {
		t.Fatal("expected IsRunning()=true after StartRun")
	}

	// Subscribe.
	ch, subID, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("unexpected Subscribe error: %v", err)
	}
	if subID == "" {
		t.Fatal("expected non-empty subscriber ID")
	}

	// Emit a few chunks.
	b.Emit(runID, "output", "hello")
	b.Emit(runID, "output", "world")
	b.Emit(runID, "status", "processing")

	// Read them back.
	chunks := readNChunks(t, ch, 3, time.Second)

	// Verify contents.
	if chunks[0].Type != "output" || chunks[0].Data != "hello" || chunks[0].Seq != 0 {
		t.Fatalf("unexpected chunk[0]: %+v", chunks[0])
	}
	if chunks[1].Type != "output" || chunks[1].Data != "world" || chunks[1].Seq != 1 {
		t.Fatalf("unexpected chunk[1]: %+v", chunks[1])
	}
	if chunks[2].Type != "status" || chunks[2].Data != "processing" || chunks[2].Seq != 2 {
		t.Fatalf("unexpected chunk[2]: %+v", chunks[2])
	}

	// All chunks should carry the correct RunID.
	for i, c := range chunks {
		if c.RunID != runID {
			t.Fatalf("chunk[%d]: expected RunID=%q, got %q", i, runID, c.RunID)
		}
	}

	// Finish the run.
	b.FinishRun(runID)

	if b.IsRunning(runID) {
		t.Fatal("expected IsRunning()=false after FinishRun")
	}

	// Read the "done" chunk.
	doneChunks := readNChunks(t, ch, 1, time.Second)
	if len(doneChunks) != 1 {
		t.Fatalf("expected 1 done chunk, got %d", len(doneChunks))
	}
	if doneChunks[0].Type != "done" {
		t.Fatalf("expected chunk type 'done', got %q", doneChunks[0].Type)
	}

	// Channel should now be closed.
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after FinishRun")
	}
}

// ---------------------------------------------------------------------------
// Multiple subscribers receive identical chunks
// ---------------------------------------------------------------------------

func TestMultipleSubscribersReceiveSameChunks(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-multi-sub"

	b.StartRun(runID)

	ch1, id1, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("sub1 Subscribe error: %v", err)
	}
	ch2, id2, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("sub2 Subscribe error: %v", err)
	}

	if id1 == id2 {
		t.Fatal("expected different subscriber IDs")
	}

	// Ensure IDs are not empty strings (they are UUIDs).
	if id1 == "" || id2 == "" {
		t.Fatal("expected non-empty subscriber IDs")
	}

	// Emit some chunks.
	b.Emit(runID, "output", "alpha")
	b.Emit(runID, "output", "beta")

	// Both should receive identical data.
	chunks1 := readNChunks(t, ch1, 2, time.Second)
	chunks2 := readNChunks(t, ch2, 2, time.Second)

	for i := 0; i < 2; i++ {
		if chunks1[i].Type != chunks2[i].Type || chunks1[i].Data != chunks2[i].Data || chunks1[i].Seq != chunks2[i].Seq {
			t.Fatalf("subscriber mismatch at index %d: %+v vs %+v", i, chunks1[i], chunks2[i])
		}
	}

	b.FinishRun(runID)
}

// ---------------------------------------------------------------------------
// Late subscriber receives the replay buffer
// ---------------------------------------------------------------------------

func TestLateSubscriberReceivesReplayBuffer(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-late-sub"

	b.StartRun(runID)

	// Early subscriber.
	ch1, _, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("sub1 Subscribe error: %v", err)
	}

	// Emit chunks before the late subscriber joins.
	b.Emit(runID, "output", "chunk-1")
	b.Emit(runID, "output", "chunk-2")
	b.Emit(runID, "output", "chunk-3")

	// Drain sub1.
	_ = readNChunks(t, ch1, 3, time.Second)

	// Late subscriber — should receive replayed chunks.
	ch2, _, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("sub2 Subscribe error: %v", err)
	}

	replay := readNChunks(t, ch2, 3, time.Second)
	if len(replay) != 3 {
		t.Fatalf("expected 3 replayed chunks, got %d", len(replay))
	}
	if replay[0].Data != "chunk-1" || replay[1].Data != "chunk-2" || replay[2].Data != "chunk-3" {
		t.Fatalf("unexpected replay content: %+v", replay)
	}

	// Emit more — both subscribers should get new chunks.
	b.Emit(runID, "output", "chunk-4")

	got1 := readNChunks(t, ch1, 1, time.Second)
	got2 := readNChunks(t, ch2, 1, time.Second)

	if len(got1) != 1 || got1[0].Data != "chunk-4" {
		t.Fatal("sub1 did not receive chunk-4")
	}
	if len(got2) != 1 || got2[0].Data != "chunk-4" {
		t.Fatal("sub2 did not receive chunk-4")
	}

	b.FinishRun(runID)
}

// ---------------------------------------------------------------------------
// Unsubscribe removes a subscriber and closes its channel
// ---------------------------------------------------------------------------

func TestUnsubscribe(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-unsub"

	b.StartRun(runID)

	ch1, id1, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("sub1 Subscribe error: %v", err)
	}
	ch2, _, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("sub2 Subscribe error: %v", err)
	}

	// Emit initial chunks.
	b.Emit(runID, "output", "before-unsub")

	// Drain both.
	_ = readNChunks(t, ch1, 1, time.Second)
	_ = readNChunks(t, ch2, 1, time.Second)

	// Unsubscribe subscriber 1.
	b.Unsubscribe(runID, id1)

	// ch1 should be closed.
	_, ok := <-ch1
	if ok {
		t.Fatal("expected ch1 to be closed after Unsubscribe")
	}

	// Emit more — only subscriber 2 should receive it.
	b.Emit(runID, "output", "after-unsub")

	// ch1 should yield nothing (it's closed, so read returns immediately).
	select {
	case _, ok := <-ch1:
		if ok {
			t.Fatal("expected no readable data on ch1 after unsubscribe")
		}
		// Channel is closed — this is the expected path.
	default:
		// Shouldn't reach here; <-ch1 on a closed channel doesn't block.
	}

	// ch2 should get the new chunk.
	got2 := readNChunks(t, ch2, 1, 100*time.Millisecond)
	if len(got2) != 1 || got2[0].Data != "after-unsub" {
		t.Fatalf("expected sub2 to receive 'after-unsub', got %+v", got2)
	}

	b.FinishRun(runID)
}

// ---------------------------------------------------------------------------
// Emit to a non-existent run does nothing (no panic)
// ---------------------------------------------------------------------------

func TestEmitToNonExistentRun(t *testing.T) {
	b := NewRunBroadcaster()

	// Must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Emit panicked: %v", r)
		}
	}()

	b.Emit("nonexistent", "output", "should-not-panic")
	b.FinishRun("nonexistent")
}

// ---------------------------------------------------------------------------
// FinishRun on a non-existent run does nothing (no panic)
// ---------------------------------------------------------------------------

func TestFinishRunNonExistent(t *testing.T) {
	b := NewRunBroadcaster()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FinishRun panicked: %v", r)
		}
	}()

	b.FinishRun("does-not-exist")
}

// ---------------------------------------------------------------------------
// IsRunning returns correct state
// ---------------------------------------------------------------------------

func TestIsRunning(t *testing.T) {
	b := NewRunBroadcaster()

	// Non-existent run.
	if b.IsRunning("no-such-run") {
		t.Fatal("expected false for non-existent run")
	}

	// After StartRun.
	b.StartRun("run-a")
	if !b.IsRunning("run-a") {
		t.Fatal("expected true after StartRun")
	}

	// After FinishRun.
	b.FinishRun("run-a")
	if b.IsRunning("run-a") {
		t.Fatal("expected false after FinishRun")
	}

	// Start another.
	b.StartRun("run-b")
	if !b.IsRunning("run-b") {
		t.Fatal("expected true after StartRun for run-b")
	}
	b.FinishRun("run-b")
	if b.IsRunning("run-b") {
		t.Fatal("expected false after FinishRun for run-b")
	}
}

// ---------------------------------------------------------------------------
// FinishRun called twice is idempotent (no panic, no deadlock)
// ---------------------------------------------------------------------------

func TestFinishRunTwice(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-finish-twice"

	b.StartRun(runID)
	b.FinishRun(runID)
	// Second FinishRun must not panic.
	b.FinishRun(runID)
}

// ---------------------------------------------------------------------------
// Subscribe to a finished run returns an error
// ---------------------------------------------------------------------------

func TestSubscribeToFinishedRun(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-sub-finished"

	b.StartRun(runID)
	b.Emit(runID, "output", "some data")
	b.FinishRun(runID)

	_, _, err := b.Subscribe(runID)
	if err == nil {
		t.Fatal("expected error when subscribing to a finished run")
	}
}

// ---------------------------------------------------------------------------
// Duplicate StartRun is a no-op
// ---------------------------------------------------------------------------

func TestStartRunNoopOnDuplicate(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-duplicate"

	b.StartRun(runID)
	b.StartRun(runID) // should be no-op

	if !b.IsRunning(runID) {
		t.Fatal("expected run to be running")
	}

	ch, _, err := b.Subscribe(runID)
	if err != nil {
		t.Fatalf("unexpected Subscribe error: %v", err)
	}

	b.Emit(runID, "output", "data")
	chunks := readNChunks(t, ch, 1, time.Second)
	if len(chunks) != 1 || chunks[0].Data != "data" {
		t.Fatal("expected to receive emitted data after duplicate StartRun")
	}

	b.FinishRun(runID)
}

// ---------------------------------------------------------------------------
// Unsubscribe on non-existent run/subscriber does nothing (no panic)
// ---------------------------------------------------------------------------

func TestUnsubscribeNonExistent(t *testing.T) {
	b := NewRunBroadcaster()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Unsubscribe panicked: %v", r)
		}
	}()

	b.Unsubscribe("no-such-run", "nonexistent-sub")
	b.Unsubscribe("no-such-run", "another-nonexistent")
}

// ---------------------------------------------------------------------------
// StartRun + Subscribe concurrency safety (quick smoke test)
// ---------------------------------------------------------------------------

func TestConcurrentStartAndSubscribe(t *testing.T) {
	b := NewRunBroadcaster()
	runID := "test-concurrent"

	b.StartRun(runID)

	const numSubs = 20
	chs := make([]<-chan OutputChunk, numSubs)

	// Subscribe from multiple goroutines.
	done := make(chan struct{})
	for i := 0; i < numSubs; i++ {
		go func(idx int) {
			ch, _, err := b.Subscribe(runID)
			if err != nil {
				t.Errorf("sub %d error: %v", idx, err)
				return
			}
			chs[idx] = ch
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < numSubs; i++ {
		<-done
	}

	// Emit from another goroutine.
	go func() {
		for i := 0; i < 5; i++ {
			b.Emit(runID, "output", "concurrent-data")
		}
		b.FinishRun(runID)
	}()

	// All subscribers should receive the chunks + done.
	for i, ch := range chs {
		if ch == nil {
			continue
		}
		all := readAllChunks(t, ch, 2*time.Second)
		if len(all) == 0 {
			t.Errorf("subscriber %d received no chunks", i)
		}
	}
}
