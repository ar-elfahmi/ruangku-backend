package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
	"ruangku-backend/app/service"
)

// Producer mengirim task ke Redis. Close dipanggil saat API berhenti.
type Producer interface {
	service.NotificationEnqueuer
	Close() error
}

// taskClient adalah bagian *asynq.Client yang dipakai producer, dipisah agar
// producer bisa dites tanpa Redis.
type taskClient interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
	Close() error
}

type producer struct {
	client taskClient
	cfg    Config
}

// NewProducer membuat producer. asynq.NewClient tidak membuka koneksi sampai
// task pertama dikirim, jadi API tetap bisa start walau Redis belum menyala.
func NewProducer(cfg Config) Producer {
	return newProducer(asynq.NewClient(cfg.RedisOpt()), cfg)
}

func newProducer(client taskClient, cfg Config) *producer {
	return &producer{client: client, cfg: cfg}
}

// EnqueueBookingStatus memasukkan task ke antrean. Batas waktu mengikuti ctx
// dari pemanggil (EnqueueContext). Task ID yang sudah ada dianggap sukses:
// task itu memang sudah ada di antrean.
func (p *producer) EnqueueBookingStatus(ctx context.Context, e model.BookingStatusEvent) error {
	task, opts, err := newBookingStatusTask(e, p.cfg)
	if err != nil {
		return err
	}
	if _, err := p.client.EnqueueContext(ctx, task, opts...); err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			return nil
		}
		return fmt.Errorf("enqueue notifikasi booking %d (%s): %w", e.BookingID, e.Status, err)
	}
	return nil
}

func (p *producer) Close() error {
	return p.client.Close()
}

var _ Producer = (*producer)(nil)
