package wal

import (
	"context"
	"log"
	"time"
)

type Queue[T Item] struct {
	ctx context.Context

	sink Sink[T]
	log  Log[T]

	// position of the _single_ reading (tailing) routine
	// MUST NOT be accessed by anything other than the q.Run() tail loop
	readIdx Idx
	// position of the last concurrent write
	writeIdx AtomicIdx
}

func NewQueue[T Item](ctx context.Context, sink Sink[T], log Log[T]) (*Queue[T], error) {
	queue := Queue[T]{
		ctx:  ctx,
		sink: sink,
		log:  log,
	}

	return &queue, nil
}

// Run continually attempts to fetch the next item from the wal and forwards it to the sink
func (q *Queue[T]) Run() {
	for {
		if q.done() {
			break
		}

		if q.size() == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		item, err := q.next()
		if err != nil {
			log.Printf("read: %s", err)
			continue
		}

		// TODO: continue at some point?
		for {
			if q.done() {
				break
			}

			err := q.sink.Handle(q.ctx, item)
			if err != nil {
				log.Printf("send %d: %s", item.idx, err)
			} else {
				break
			}
		}
	}
}

func (q *Queue[T]) Add(item *T) error {
	return q.log.Write(q.writeIdx.Add(1), item)
}

func (q *Queue[T]) Close() error {
	if err := TryFlush(q.ctx, q.sink); err != nil {
		return err
	}

	return q.log.Close()
}

func (q *Queue[T]) done() bool {
	select {
	case <-q.ctx.Done():
		return true
	default:
		return false
	}
}

func (q *Queue[T]) size() int {
	return int(q.writeIdx.Load() - q.readIdx)
}

func (q *Queue[T]) next() (*sinkItem[T], error) {
	idx := q.readIdx
	q.readIdx++

	item, err := q.log.Read(idx)
	if err != nil {
		return nil, err
	}

	si := sinkItem[T]{
		data: item,
		idx:  idx,
		drp:  q.log,
	}
	return &si, nil
}

type sinkItem[T Item] struct {
	data *T
	idx  Idx
	drp  Dropper
}

func (s *sinkItem[T]) Close() error {
	return s.drp.Drop(s.idx)
}

func (s *sinkItem[T]) Data() *T {
	return s.data
}
