package wal

import (
	"context"
	"log"
	"sync"
	"time"
)

// Queue is a write-ahead "queue"
//   - Items are added by synchronously added by producers.
//   - Items are stored permanently until consumed, that means across application
//     and system crashes
//   - Items are asynchronously forwarded to a Sink
type Queue[T Item] struct {
	ctx  context.Context
	sink Sink[T]
	chk  Checker

	log Log[T]

	mu       sync.RWMutex
	readIdx  Idx
	writeIdx Idx
	truncIdx Idx

	truncFreq time.Duration
}

type Opts struct {
	Dir       string
	BufCount  int
	TruncFreq time.Duration

	Checker Checker
}

func NewQueue[T Item](ctx context.Context, sink Sink[T], opts Opts) (*Queue[T], error) {
	bufSink := NewBufSink(sink, opts.BufCount)
	chk := opts.Checker
	if chk == nil {
		chk = LogChecker{}
	}

	log, err := NewLog[T](opts.Dir)
	if err != nil {
		return nil, err
	}

	queue := Queue[T]{
		ctx:  ctx,
		sink: bufSink,
		chk:  chk,
		log:  log,

		truncFreq: opts.TruncFreq,
	}

	return &queue, nil
}

func (q *Queue[T]) Run() {
	go func() {
		tick := time.NewTicker(q.truncFreq)
		defer tick.Stop()
		for range tick.C {
			select {
			case <-q.ctx.Done():
				break
			}

			if err := q.truncate(); q.chk.TruncFail(err) {
				break
			}
		}
	}()

	for {
		// end if canceled
		select {
		case <-q.ctx.Done():
			break
		}

		if q.Size() == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		item, ack := q.next()
		if item == nil {
			break
		}

		if err := q.sink.Handle(item, ack); q.chk.SinkFail(err) {
			break
		}
	}
}

func (q *Queue[T]) Add(item *T) error {
	q.assert()
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.log.Write(q.writeIdx, item); err != nil {
		return err
	}
	q.writeIdx++
	return nil
}

func (q *Queue[T]) next() (*T, Ack) {
	q.assert()
	q.mu.RLock()
	defer q.mu.RUnlock()

	item, err := q.log.Read(q.readIdx)
	if q.chk.ReadFail(err) {
		return nil, nil
	}
	idx := q.readIdx
	q.readIdx++

	ack := func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.truncIdx = idx
	}
	return item, ack
}

func (q *Queue[T]) Size() int {
	q.assert()
	q.mu.RLock()
	defer q.mu.RUnlock()

	return int(q.writeIdx - q.readIdx)
}

func (q *Queue[T]) truncate() error {
	if flusher, ok := q.sink.(Flusher); ok {
		if err := flusher.Flush(); err != nil {
			return err
		}
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	return q.log.Truncate(q.truncIdx + 1)
}

func (q *Queue[T]) assert() {
	if q.readIdx > q.writeIdx {
		panic("readIdx must <= writeIdx")
	}
}

func (q *Queue[T]) Close() error {
	if err := q.truncate(); err != nil {
		return err
	}
	return q.log.Close()
}

// Checker determines whether a failed operation is fatal
//
// Depending on the return value, wal tailing is either terminated (true) or the
// current item is skipped (false)
type Checker interface {
	// Reading from the log failed
	ReadFail(err error) bool
	// Sending to the sink failed
	SinkFail(err error) bool
	// Truncating failed
	TruncFail(err error) bool
}

type LogChecker struct{}

func (l LogChecker) ReadFail(err error) bool {
	log.Println("failed reading from the wal:", err)
	return false
}

func (l LogChecker) SinkFail(err error) bool {
	log.Println("failed sending:", err)
	return false
}

func (l LogChecker) TruncFail(err error) bool {
	log.Println("failed truncating:", err)
	return true
}
