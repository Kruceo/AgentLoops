// Package runs provides domain types and components for managing task run output,
// including an in-memory pub/sub broadcaster for streaming output chunks to
// multiple subscribers (e.g., SSE clients).
package runs

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

// OutputChunk represents a single chunk of output from a run execution.
// It is used by the RunBroadcaster to stream output to subscribers.
type OutputChunk struct {
	RunID string `json:"runId"`
	Type  string `json:"type"` // "output" | "status" | "error" | "done"
	Data  string `json:"data"`
	Seq   int    `json:"seq"` // monotonically increasing sequence number
}

const (
	// chanBufSize is the buffer size for each subscriber channel.
	// This prevents slow consumers from blocking the emitter.
	chanBufSize = 256

	// maxBufChunks is the maximum number of chunks kept in the replay buffer.
	// When exceeded, oldest chunks are dropped to limit memory usage.
	maxBufChunks = 10000
)

// runStream holds the per-run state: subscriber channels, replay buffer,
// completion flag, and sequence counter.
type runStream struct {
	mu     sync.Mutex
	chans  map[string]chan OutputChunk // subscriber channels keyed by subscriber ID
	buffer []OutputChunk               // replay buffer for late subscribers
	done   bool                        // whether the run has finished
	seq    int                         // per-run monotonically increasing sequence counter
}

// RunBroadcaster is an in-memory pub/sub component that allows multiple
// clients to subscribe to the output chunks of a running task (by runID).
// It also maintains a replay buffer so that late subscribers can catch up
// with previously emitted chunks.
//
// All methods are thread-safe. The broadcaster is designed for a single
// producer (the task runner) and multiple consumers (SSE clients).
type RunBroadcaster struct {
	mu      sync.RWMutex
	streams map[string]*runStream
}

// NewRunBroadcaster creates a new empty RunBroadcaster.
func NewRunBroadcaster() *RunBroadcaster {
	return &RunBroadcaster{
		streams: make(map[string]*runStream),
	}
}

// StartRun registers a new run stream. If a stream for the given runID
// already exists, this is a no-op.
func (rb *RunBroadcaster) StartRun(runID string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if _, ok := rb.streams[runID]; ok {
		return
	}

	rb.streams[runID] = &runStream{
		chans:  make(map[string]chan OutputChunk),
		buffer: make([]OutputChunk, 0, 1024),
	}
}

// Subscribe subscribes to a run's output. It returns:
//   - A read-only channel that receives OutputChunks
//   - A subscriber ID (used for Unsubscribe)
//   - An error if the runID does not exist as an active stream
//
// The new subscriber immediately receives all previously emitted chunks
// (the replay buffer) before any future chunks. The channel has a buffer
// of 256; if a subscriber is too slow its messages will be skipped with
// a warning log (the emitter never blocks).
//
// If the run has already finished (done flag set), Subscribe returns an error.
func (rb *RunBroadcaster) Subscribe(runID string) (<-chan OutputChunk, string, error) {
	rb.mu.RLock()
	stream, ok := rb.streams[runID]
	rb.mu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("run %s not found", runID)
	}

	subID := uuid.NewString()
	ch := make(chan OutputChunk, chanBufSize)

	stream.mu.Lock()

	// Check if the run finished between our map lookup and acquiring the lock.
	if stream.done {
		stream.mu.Unlock()
		return nil, "", fmt.Errorf("run %s has already finished", runID)
	}

	// Add channel to the map FIRST, before replay, so that FinishRun
	// (which holds stream.mu) sees our channel and can close it.
	stream.chans[subID] = ch

	// Replay all previously buffered chunks to the new subscriber.
	// Use non-blocking sends so a slow subscriber never blocks the broadcaster.
	for _, chunk := range stream.buffer {
		select {
		case ch <- chunk:
		default:
			// Drop if channel is full — subscriber will receive future chunks.
		}
	}

	stream.mu.Unlock()

	return ch, subID, nil
}

// Unsubscribe removes a subscriber and closes its channel. If the subscriber
// or run does not exist, this is a no-op.
func (rb *RunBroadcaster) Unsubscribe(runID string, subID string) {
	rb.mu.RLock()
	stream, ok := rb.streams[runID]
	rb.mu.RUnlock()
	if !ok {
		return
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if ch, ok := stream.chans[subID]; ok {
		close(ch)
		delete(stream.chans, subID)
	}
}

// Emit broadcasts an OutputChunk to all subscribers of the given run.
// It creates a chunk with the provided type and data, assigns it a
// monotonically increasing sequence number, appends it to the replay
// buffer, and fans it out to every subscriber channel.
//
// If a subscriber's channel buffer is full, the subscriber is skipped
// and a warning is logged. The emitter never blocks.
//
// If the runID does not exist or the run has finished, Emit returns
// silently (no-op).
func (rb *RunBroadcaster) Emit(runID string, chunkType string, data string) {
	rb.mu.RLock()
	stream, ok := rb.streams[runID]
	rb.mu.RUnlock()
	if !ok {
		return
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.done {
		return
	}

	chunk := OutputChunk{
		RunID: runID,
		Type:  chunkType,
		Data:  data,
		Seq:   stream.seq,
	}
	stream.seq++

	// Append to the replay buffer. If over capacity, drop the oldest chunks.
	stream.buffer = append(stream.buffer, chunk)
	if len(stream.buffer) > maxBufChunks {
		overflow := len(stream.buffer) - maxBufChunks
		stream.buffer = stream.buffer[overflow:]
	}

	// Fan out to all subscribers. Never block — skip slow consumers.
	for subID, ch := range stream.chans {
		select {
		case ch <- chunk:
		default:
			log.Printf("warning: dropping chunk seq=%d for subscriber %s of run %s (channel full)", chunk.Seq, subID, runID)
		}
	}
}

// FinishRun marks the run as done, emits a "done" chunk to all subscribers,
// closes all subscriber channels, and removes the stream entry from the
// broadcaster map. If the run does not exist, this is a no-op.
//
// The stream is removed from the map first so that no new subscribers can
// attach after cleanup begins.
func (rb *RunBroadcaster) FinishRun(runID string) {
	// Remove the stream from the map first so no new subscribers can attach.
	rb.mu.Lock()
	stream, ok := rb.streams[runID]
	if !ok {
		rb.mu.Unlock()
		return
	}
	delete(rb.streams, runID)
	rb.mu.Unlock()

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.done {
		return
	}
	stream.done = true

	// Emit the final "done" chunk.
	chunk := OutputChunk{
		RunID: runID,
		Type:  "done",
		Seq:   stream.seq,
	}
	stream.seq++
	stream.buffer = append(stream.buffer, chunk)

	// Send the done chunk to all subscribers, then close their channels.
	for subID, ch := range stream.chans {
		select {
		case ch <- chunk:
		default:
			log.Printf("warning: dropping done chunk for subscriber %s of run %s (channel full)", subID, runID)
		}
		close(ch)
	}

	// Clear the channels map.
	stream.chans = make(map[string]chan OutputChunk)
}

// IsRunning checks whether a run is currently active — meaning it has a
// stream entry and has not been marked done. Returns false if the runID
// does not exist or the run has finished.
func (rb *RunBroadcaster) IsRunning(runID string) bool {
	rb.mu.RLock()
	stream, ok := rb.streams[runID]
	rb.mu.RUnlock()
	if !ok {
		return false
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()
	return !stream.done
}
