package wal

import (
	"io"
	"sync/atomic"

	"github.com/tidwall/wal"
)

type Item interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// Idx is a monotonically increasing index to entries in the Log
type Idx = uint64
type AtomicIdx = atomic.Uint64

// Log is a write-ahead log on disk
type Log[T Item] interface {
	io.Closer

	// Write / Creates an item at Idx
	Write(Idx, *T) error
	// Read the item at the given index
	Read(Idx) (*T, error)
	Dropper
}

type Dropper interface {
	Drop(Idx) error
}

func NewLog[T Item](dir string) (Log[T], error) {
	log, err := wal.Open(dir, &wal.Options{})
	if err != nil {
		return nil, err
	}

	return &walLog[T]{log: *log}, nil
}

type walLog[T Item] struct {
	log wal.Log
}

func (w *walLog[T]) Read(idx Idx) (*T, error) {
	data, err := w.log.Read(idx)
	if err != nil {
		return nil, err
	}

	var t T
	if err := t.Unmarshal(data); err != nil {
		return nil, err
	}
	return &t, nil
}

func (w *walLog[T]) Write(idx Idx, item *T) error {
	data, err := (*item).Marshal()
	if err != nil {
		return err
	}

	return w.log.Write(idx, data)
}

func (w *walLog[T]) Drop(idx Idx) error {
	// NOTE: this is sematically not exactly equivalent, but Drop() is expected
	// to be called in monotonically increasing order.
	return w.log.TruncateFront(idx)
}

func (w *walLog[T]) Close() error {
	return w.Close()
}
