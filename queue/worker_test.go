package queue

import (
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestNewWorkerDoesNotConnectToRedis(t *testing.T) {
	// Alamat yang pasti mati: membuat worker tidak boleh membuka koneksi.
	w := NewWorker(Config{RedisAddr: "127.0.0.1:1", Queue: "it-queue", Concurrency: 2}, &fakeStore{})
	if w == nil {
		t.Fatal("NewWorker mengembalikan nil")
	}
	w.Shutdown()
}

func TestWorkerRunAfterShutdownFails(t *testing.T) {
	w := NewWorker(Config{RedisAddr: "127.0.0.1:1", Concurrency: 1}, &fakeStore{})
	w.Shutdown()

	done := make(chan error, 1)
	go func() { done <- w.Run() }()

	select {
	case err := <-done:
		if !errors.Is(err, ErrWorkerStopped) {
			t.Fatalf("Run setelah Shutdown harus ErrWorkerStopped, dapat %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run setelah Shutdown tidak boleh menggantung")
	}
}

func TestWorkerShutdownIsIdempotent(t *testing.T) {
	w := NewWorker(Config{RedisAddr: "127.0.0.1:1", Concurrency: 1}, &fakeStore{})
	w.Shutdown()
	w.Shutdown() // tidak boleh panic (close dua kali)
}

func TestWorkerQueueDefaultsAndConcurrencyFloor(t *testing.T) {
	cfg := workerConfig(Config{})
	if cfg.Concurrency != defaultConcurrency {
		t.Fatalf("concurrency nol harus jatuh ke nilai positif, dapat %d", cfg.Concurrency)
	}
	if _, ok := cfg.Queues[DefaultQueue]; !ok || len(cfg.Queues) != 1 {
		t.Fatalf("worker harus hanya membaca queue %s: %v", DefaultQueue, cfg.Queues)
	}

	custom := workerConfig(Config{Queue: "it-x", Concurrency: 4})
	if _, ok := custom.Queues["it-x"]; !ok || custom.Concurrency != 4 {
		t.Fatalf("konfigurasi kustom tidak dipakai: %+v", custom)
	}
}

func TestWorkerConfigUsesCustomRetryDelay(t *testing.T) {
	custom := func(n int, err error, task *asynq.Task) time.Duration { return 123 * time.Millisecond }

	cfg := workerConfig(Config{RetryDelay: custom})

	if cfg.RetryDelayFunc == nil || cfg.RetryDelayFunc(1, nil, nil) != 123*time.Millisecond {
		t.Fatal("RetryDelay kustom harus dipakai")
	}
	if workerConfig(Config{}).RetryDelayFunc != nil {
		t.Fatal("tanpa RetryDelay, Asynq memakai backoff eksponensial bawaannya")
	}
}
