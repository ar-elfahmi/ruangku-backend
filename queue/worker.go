package queue

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/hibiken/asynq"
)

// ErrWorkerStopped dikembalikan Run bila Shutdown sudah dipanggil lebih dulu.
var ErrWorkerStopped = errors.New("worker sudah dihentikan")

// Worker memproses task notifikasi dari Redis. Ia berjalan di proses
// terpisah dari API (cmd/worker) dan hanya butuh Redis dan penyimpanan
// notifikasi, tidak butuh PostgreSQL.
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux

	stop chan struct{}
	once sync.Once
}

// NewWorker menyiapkan worker. Belum ada koneksi ke Redis sebelum Run.
func NewWorker(cfg Config, store NotificationStore) *Worker {
	p := &processor{store: store}
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskTypeBookingStatus, p.ProcessBookingStatus)

	return &Worker{
		server: asynq.NewServer(cfg.RedisOpt(), workerConfig(cfg)),
		mux:    mux,
		stop:   make(chan struct{}),
	}
}

// workerConfig memetakan Config ke konfigurasi server Asynq: hanya membaca
// queue notifikasi, dengan jumlah worker sesuai WORKER_CONCURRENCY. Retry
// memakai backoff eksponensial bawaan Asynq.
func workerConfig(cfg Config) asynq.Config {
	queueName := cfg.Queue
	if queueName == "" {
		queueName = DefaultQueue
	}
	concurrency := cfg.Concurrency
	if concurrency < 1 {
		concurrency = defaultConcurrency
	}
	return asynq.Config{
		Concurrency:    concurrency,
		Queues:         map[string]int{queueName: 1},
		RetryDelayFunc: cfg.RetryDelay,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			// Payload tidak dicatat; cukup tipe task dan penyebabnya.
			log.Printf("worker: task %s gagal: %v", task.Type(), err)
		}),
	}
}

// Run memulai pemrosesan dan menunggu sampai Shutdown dipanggil. Error
// dikembalikan bila server gagal start (misalnya Redis tidak terjangkau).
func (w *Worker) Run() error {
	select {
	case <-w.stop:
		return ErrWorkerStopped
	default:
	}
	if err := w.server.Start(w.mux); err != nil {
		return err
	}
	<-w.stop
	return nil
}

// Shutdown berhenti menerima task baru dan menunggu task yang sedang
// berjalan selesai (sampai batas waktu Asynq). Aman dipanggil berulang kali.
func (w *Worker) Shutdown() {
	w.once.Do(func() {
		close(w.stop)
		w.server.Shutdown()
	})
}
