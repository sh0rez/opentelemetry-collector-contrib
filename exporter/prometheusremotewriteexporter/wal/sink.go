package wal

import (
	"context"
	"io"
)

// Sink receives the requests read from the write-ahead log an shall forward
// them a remote-write receiver
type Sink[T any] interface {
	Handle(context.Context, SinkItem[T]) error
}

type SinkItem[T any] interface {
	Data() *T

	// Sinks must Close() the SinkItem once sent for cleanup (e.g. remove from disk)
	io.Closer
}

type FnSink[T any] func(context.Context, SinkItem[T]) error

func (fn FnSink[T]) Handle(ctx context.Context, item SinkItem[T]) error {
	return fn(ctx, item)
}

// Flusher is a special type of Sink that may keep in-memory state that should
// be persisted
type Flusher interface {
	Flush(context.Context) error
}

// TryFlush attempts to flush the sink, if supported
func TryFlush(ctx context.Context, sink any) error {
	if fl, ok := sink.(Flusher); ok {
		return fl.Flush(ctx)
	}
	return nil
}

// BufSink adds buffering to an existing sink implementation, by collecting a
// number of items in memory before forwarding to the actual sink
type BufSink[T any] struct {
	sink Sink[T]
	buf  []SinkItem[T]
}

func NewBufSink[T any](sink Sink[T], cap int) *BufSink[T] {
	return &BufSink[T]{
		sink: sink,
		buf:  make([]SinkItem[T], 0, cap),
	}
}

func (b *BufSink[T]) Handle(ctx context.Context, item SinkItem[T]) error {
	if len(b.buf) == cap(b.buf) {
		if err := b.Flush(ctx); err != nil {
			return err
		}
	}

	b.buf = append(b.buf, item)
	return nil
}

func (b *BufSink[T]) Flush(ctx context.Context) error {
	// send buffered items to sink
	for _, item := range b.buf {
		if err := b.sink.Handle(ctx, item); err != nil {
			return err
		}
	}

	// attempt to flush any following sinks
	if err := TryFlush(ctx, b.sink); err != nil {
		return err
	}

	b.buf = b.buf[:0]
	return nil
}
