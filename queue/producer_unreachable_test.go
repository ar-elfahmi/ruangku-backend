package queue

import (
	"context"
	"testing"
	"time"
)

// Redis tidak terjangkau: producer harus gagal dengan error dalam batas waktu
// context, tidak menggantung, sehingga hook booking bisa mencatatnya dan
// response booking tetap 2xx. Alamat 127.0.0.1:1 pasti menolak koneksi.
func TestProducerRedisUnreachableFailsWithinTimeLimit(t *testing.T) {
	p := NewProducer(Config{RedisAddr: "127.0.0.1:1", Queue: "it-unreachable"})
	t.Cleanup(func() { _ = p.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := p.EnqueueBookingStatus(ctx, sampleEvent)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Redis mati harus menghasilkan error, bukan sukses")
	}
	if elapsed > 4*time.Second {
		t.Fatalf("producer menggantung %s, melewati batas waktu context", elapsed)
	}
}
