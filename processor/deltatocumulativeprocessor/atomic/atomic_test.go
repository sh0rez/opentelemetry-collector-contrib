package atomic_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"iter"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"runtime/pprof"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/exp/metrics/identity"
	"github.com/puzpuzpuz/xsync/v3"
)

var keep any

func Benchmark(b *testing.B) {
	cases := []struct {
		state string
		value string
		run   func([]identity.Stream) func(iter.Seq2[identity.Stream, int64])
	}{
		{
			state: "prealloc", value: "atomic",
			run: use_prealloc[atomic.Int64],
		},
		{
			state: "prealloc", value: "mutex",
			run: use_prealloc[MutexInt64],
		},
		{
			state: "mutex", value: "atomic",
			run: use_mutex[atomic.Int64],
		},
		{
			state: "mutex", value: "mutex",
			run: use_mutex[MutexInt64],
		},
		{
			state: "syncmap", value: "atomic",
			run: use_syncmap[atomic.Int64],
		},
		{
			state: "syncmap", value: "mutex",
			run: use_syncmap[MutexInt64],
		},
		{
			state: "xsync", value: "atomic",
			run: use_xsync[atomic.Int64],
		},
		{
			state: "xsync", value: "mutex",
			run: use_xsync[MutexInt64],
		},
	}

	var (
		lastN   int
		profile bytes.Buffer
		total   int
	)

	for _, cs := range cases {
		b.Run(fmt.Sprintf("state=%s/value=%s", cs.state, cs.value), func(b *testing.B) {
			ids := make([]identity.Stream, max(1, b.N/10))
			for i := range ids {
				ids[i] = itoid(i)
			}

			iter := func(n int) iter.Seq2[identity.Stream, int64] {
				cids := ids[:0]
				return func(yield func(identity.Stream, int64) bool) {
					for i := range n {
						if i%10 == 0 {
							cids = ids[:len(cids)+1]
						}
						yield(cids[i%len(cids)], int64(i))
					}
				}
			}

			var wg sync.WaitGroup
			run := cs.run(ids)

			P := runtime.GOMAXPROCS(0)
			bs := make([]int, P)
			for i := range bs {
				bs[i] = b.N / P
				if i == 0 {
					bs[i] = b.N % P
				}
			}

			pprof.StartCPUProfile(&profile)

			b.ResetTimer()
			b.ReportAllocs()

			for _, bs := range bs {
				wg.Add(1)
				go func() {
					run(iter(bs))
					wg.Done()
				}()
			}

			wg.Wait()

			pprof.StopCPUProfile()
			os.WriteFile(fmt.Sprintf("cpu/%s-%s-%d.pprof", cs.state, cs.value, total), profile.Bytes(), 0644)
			profile.Reset()
			if b.N < lastN {
				total++
			}
			lastN = b.N
		})
	}
}

type pval[T any] interface {
	*T
	Add(delta int64) (new int64)
}

func use_prealloc[V any, P pval[V]](ids []identity.Stream) func(iter.Seq2[identity.Stream, int64]) {
	m := make(map[identity.Stream]P)
	for _, id := range ids {
		m[id] = new(V)
	}
	return func(iter iter.Seq2[identity.Stream, int64]) {
		for id, dp := range iter {
			ptr := m[id]
			ptr.Add(dp)
		}
		keep = m
	}
}

func use_syncmap[V any, P pval[V]]([]identity.Stream) func(iter.Seq2[identity.Stream, int64]) {
	m := new(sync.Map)
	return func(iter iter.Seq2[identity.Stream, int64]) {
		zero := new(V)
		for id, dp := range iter {
			ptr, ok := m.LoadOrStore(id, zero)
			if !ok {
				zero = new(V)
			}
			ptr.(*atomic.Int64).Add(dp)
		}
		keep = m
	}
}

func use_xsync[V any, P pval[V]]([]identity.Stream) func(iter.Seq2[identity.Stream, int64]) {
	m := xsync.NewMapOf[identity.Stream, P]()
	return func(iter iter.Seq2[identity.Stream, int64]) {
		zero := new(V)
		for id, dp := range iter {
			ptr, ok := m.LoadOrStore(id, zero)
			if !ok {
				zero = new(V)
			}
			ptr.Add(dp)
		}
		keep = m
	}
}

func use_mutex[V any, P pval[V]]([]identity.Stream) func(iter.Seq2[identity.Stream, int64]) {
	var mtx sync.RWMutex
	m := make(map[identity.Stream]*atomic.Int64)
	return func(iter iter.Seq2[identity.Stream, int64]) {
		for id, dp := range iter {
			mtx.RLock()
			at, ok := m[id]
			mtx.RUnlock()
			if !ok {
				mtx.Lock()
				at = new(atomic.Int64)
				m[id] = at
				mtx.Unlock()
			}
			at.Add(dp)
		}
		keep = m
	}
}

type MutexInt64 struct {
	mtx sync.Mutex
	v   int64
}

func (m *MutexInt64) Add(delta int64) int64 {
	m.mtx.Lock()
	m.v += delta
	out := m.v
	m.mtx.Unlock()
	return out
}

func itoid(i int) identity.Stream {
	type streamid struct {
		identity.Metric
		attr [16]byte
	}

	var id streamid
	binary.LittleEndian.PutUint64(id.attr[:], uint64(i))
	return *(*identity.Stream)(unsafe.Pointer(&id))
}
