//go:build integration

package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/testutil"
)

// switchableStore meniru penyimpanan notifikasi yang bisa dimatikan dan
// dinyalakan lagi, dengan perilaku unik (booking_id, status) seperti MongoDB.
type switchableStore struct {
	mu       sync.Mutex
	failing  bool
	accepted []model.Notification
}

func (s *switchableStore) InsertNotification(ctx context.Context, n model.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing {
		return context.DeadlineExceeded // meniru MongoDB tidak terjangkau
	}
	for _, existing := range s.accepted {
		if existing.BookingID == n.BookingID && existing.Status == n.Status {
			return repository.ErrDuplicate
		}
	}
	s.accepted = append(s.accepted, n)
	return nil
}

func (s *switchableStore) setFailing(failing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failing = failing
}

func (s *switchableStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.accepted)
}

// integrationConfig memakai queue unik di Redis DB test supaya test tidak
// bertabrakan dengan worker lain. Queue tidak dihapus setelahnya.
func integrationConfig(env testutil.Env) Config {
	return Config{
		RedisAddr:   env.RedisAddr,
		RedisDB:     env.RedisDB,
		Queue:       testutil.Unique("queue"),
		Concurrency: 1,
	}
}

func integrationEvent() model.BookingStatusEvent {
	return model.BookingStatusEvent{
		BookingID:  int(time.Now().UnixNano()%1_000_000_000) + 1,
		OwnerID:    1,
		Status:     model.StatusPending,
		OccurredAt: time.Now(),
	}
}

func enqueue(t *testing.T, cfg Config, e model.BookingStatusEvent) {
	t.Helper()
	p := NewProducer(cfg)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.EnqueueBookingStatus(ctx, e); err != nil {
		t.Fatalf("enqueue gagal (Redis test menyala?): %v", err)
	}
}

func startWorker(t *testing.T, cfg Config, store NotificationStore) *Worker {
	t.Helper()
	w := NewWorker(cfg, store)
	go func() { _ = w.Run() }()
	t.Cleanup(w.Shutdown)
	return w
}

// B-I6: worker mati. Task tetap pending di Redis; setelah worker baru
// menyala, task diproses tepat satu kali.
func TestWorkerStoppedTaskWaitsInRedisThenProcessedOnce(t *testing.T) {
	env := testutil.Require(t)
	cfg := integrationConfig(env)
	store := &switchableStore{}
	event := integrationEvent()

	enqueue(t, cfg, event)

	inspector := asynq.NewInspector(cfg.RedisOpt())
	defer inspector.Close()
	info, err := inspector.GetTaskInfo(cfg.Queue, TaskID(event.BookingID, event.Status))
	if err != nil {
		t.Fatalf("task tidak ditemukan di Redis: %v", err)
	}
	if info.State != asynq.TaskStatePending {
		t.Fatalf("tanpa worker, task harus pending, dapat %v", info.State)
	}
	time.Sleep(500 * time.Millisecond) // memastikan memang tidak ada yang mengambil
	if store.count() != 0 {
		t.Fatal("tidak ada worker, tetapi notifikasi tersimpan")
	}

	startWorker(t, cfg, store)

	testutil.WaitFor(t, testutil.DefaultWait, testutil.DefaultInterval, func() bool { return store.count() == 1 },
		"worker baru memproses task yang menunggu")
	time.Sleep(500 * time.Millisecond)
	if got := store.count(); got != 1 {
		t.Fatalf("harus tepat satu notifikasi, dapat %d", got)
	}
}

// B-I8: penyimpanan mati. Task masuk status retry (hitungan retry naik);
// setelah penyimpanan pulih, task diproses tepat satu kali.
func TestWorkerStoreDownRetriesThenProcessedOnce(t *testing.T) {
	env := testutil.Require(t)
	cfg := integrationConfig(env)
	cfg.RetryDelay = func(n int, err error, task *asynq.Task) time.Duration { return 300 * time.Millisecond }
	store := &switchableStore{failing: true}
	event := integrationEvent()
	taskID := TaskID(event.BookingID, event.Status)

	enqueue(t, cfg, event)
	startWorker(t, cfg, store)

	inspector := asynq.NewInspector(cfg.RedisOpt())
	defer inspector.Close()
	testutil.WaitFor(t, testutil.DefaultWait, testutil.DefaultInterval, func() bool {
		info, err := inspector.GetTaskInfo(cfg.Queue, taskID)
		return err == nil && info.Retried >= 1
	}, "task gagal dan masuk retry saat penyimpanan mati")
	if store.count() != 0 {
		t.Fatal("penyimpanan mati, tetapi notifikasi tersimpan")
	}

	store.setFailing(false)

	testutil.WaitFor(t, testutil.DefaultWait, testutil.DefaultInterval, func() bool { return store.count() == 1 },
		"task diproses setelah penyimpanan pulih")
	time.Sleep(500 * time.Millisecond)
	if got := store.count(); got != 1 {
		t.Fatalf("harus tepat satu notifikasi, dapat %d", got)
	}
}
