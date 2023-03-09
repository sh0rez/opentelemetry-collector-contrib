package wal_test

import (
	"context"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter/wal"
)

type Item struct{}

func (i Item) Marshal() ([]byte, error) {
	return nil, nil
}

func (i Item) Unmarshal([]byte) error {
	return nil
}

func TestQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sink := wal.FnSink[Item](func(item *Item, ack wal.Ack) error {
		_ = item
		if ack != nil {
			ack()
		}
		return nil
	})

	q, err := wal.NewQueue[Item](ctx, sink, wal.Opts{
		Dir:       t.TempDir(),
		BufSize:   10,
		TruncFreq: 10 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer q.Close()

	go q.Run()
}
