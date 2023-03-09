package wal

// Ack confirms that an item is safely stored and may removed from the wal
// (truncated)
type Ack func()

// Sink receives the requests read from the write-ahead log an shall forward
// them a remote-write receiver
type Sink[T any] interface {
	Handle(item *T, ack Ack) error
}

type FnSink[T any] func(*T, Ack) error

func (fn FnSink[T]) Handle(item *T, ack Ack) error {
	return fn(item, ack)
}

type Flusher interface {
	Flush() error
}

type BufSink[T any] struct {
	sink Sink[T]

	buf  []*T
	acks []Ack
}

func NewBufSink[T any](sink Sink[T], cap int) *BufSink[T] {
	return &BufSink[T]{
		sink: sink,
		buf:  make([]*T, 0, cap),
		acks: make([]Ack, 0, cap),
	}
}

func (b *BufSink[T]) Handle(item *T, ack Ack) error {
	if len(b.buf) == cap(b.buf) {
		if err := b.Flush(); err != nil {
			return err
		}
	}

	b.buf = append(b.buf, item)
	b.acks = append(b.acks, ack)
	return nil
}

func (b *BufSink[T]) Flush() error {
	for i, item := range b.buf {
		if err := b.sink.Handle(item, nil); err != nil {
			return err
		}
		b.acks[i]()
	}

	b.buf = b.buf[:0]
	b.acks = b.acks[:0]
	return nil
}
