package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProjectionBackoffDelayIsBoundedExponential(t *testing.T) {
	max := 30 * time.Second
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // capped
		{50, 30 * time.Second},
		{-1, 1 * time.Second},
	}
	for _, tt := range cases {
		if got := projectionBackoffDelay(tt.attempt, max); got != tt.want {
			t.Errorf("projectionBackoffDelay(%d, %s) = %s, want %s", tt.attempt, max, got, tt.want)
		}
	}
}

func TestMemoryStreamSourceReturnsEventsThenEnds(t *testing.T) {
	source := NewMemoryStreamSource([]StreamEvent{
		{Cursor: "1", Kind: "commit", DID: "did:plc:a", Collection: "tv.subcult.profile"},
		{Cursor: "2", Kind: "commit", DID: "did:plc:b", Collection: "tv.subcult.profile"},
	})
	ctx := t.Context()

	first, err := source.Next(ctx)
	if err != nil || first.Cursor != "1" {
		t.Fatalf("first Next() = %+v, %v", first, err)
	}
	second, err := source.Next(ctx)
	if err != nil || second.Cursor != "2" {
		t.Fatalf("second Next() = %+v, %v", second, err)
	}
	if _, err := source.Next(ctx); !errors.Is(err, ErrProjectionStreamEnded) {
		t.Fatalf("third Next() error = %v, want ErrProjectionStreamEnded", err)
	}
}

func TestMemoryStreamSourceRespectsCanceledContext(t *testing.T) {
	source := NewMemoryStreamSource([]StreamEvent{{Cursor: "1"}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := source.Next(ctx); err == nil {
		t.Fatal("Next() with canceled context: want error, got nil")
	}
}

func TestProjectionRunStopsAtStreamEnd(t *testing.T) {
	// Run() with a nil db is only reachable if ProcessEvent is never called;
	// an empty source proves the loop returns cleanly on
	// ErrProjectionStreamEnded without touching the database.
	processor := &ProjectionProcessor{}
	source := NewMemoryStreamSource(nil)
	stats, err := processor.Run(t.Context(), source, ProjectionRunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stats != (ProjectionStats{}) {
		t.Fatalf("Run() stats = %+v, want zero value", stats)
	}
}

func TestProjectionRunStopsOnCanceledContext(t *testing.T) {
	processor := &ProjectionProcessor{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	source := NewMemoryStreamSource([]StreamEvent{{Cursor: "1"}})
	stats, err := processor.Run(ctx, source, ProjectionRunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stats != (ProjectionStats{}) {
		t.Fatalf("Run() stats = %+v, want zero value", stats)
	}
}

// transientThenEndSource returns a transient error a fixed number of times
// before ending the stream, used to exercise Run's bounded-attempt give-up
// path without a real network connection or real sleeps.
type transientThenEndSource struct {
	failuresLeft int
	failWith     error
}

func (s *transientThenEndSource) Next(ctx context.Context) (StreamEvent, error) {
	if s.failuresLeft > 0 {
		s.failuresLeft--
		return StreamEvent{}, s.failWith
	}
	return StreamEvent{}, ErrProjectionStreamEnded
}

func TestProjectionRunGivesUpAfterMaxAttempts(t *testing.T) {
	processor := &ProjectionProcessor{}
	source := &transientThenEndSource{failuresLeft: 5, failWith: errors.New("transient dial error")}
	_, err := processor.Run(t.Context(), source, ProjectionRunOptions{
		MaxReconnectDelay: time.Millisecond, // keep the test fast
		MaxAttempts:       2,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want reconnect-exhausted error")
	}
}

func TestProjectionRunRecoversFromTransientErrors(t *testing.T) {
	processor := &ProjectionProcessor{}
	source := &transientThenEndSource{failuresLeft: 2, failWith: errors.New("transient dial error")}
	stats, err := processor.Run(t.Context(), source, ProjectionRunOptions{
		MaxReconnectDelay: time.Millisecond,
		MaxAttempts:       5,
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want recovery before giving up", err)
	}
	if stats != (ProjectionStats{}) {
		t.Fatalf("Run() stats = %+v, want zero value (no commit events)", stats)
	}
}
