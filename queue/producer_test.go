package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
)

type fakeClient struct {
	err    error
	tasks  []*asynq.Task
	opts   [][]asynq.Option
	closed bool
}

func (f *fakeClient) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	f.tasks = append(f.tasks, task)
	f.opts = append(f.opts, opts)
	return nil, f.err
}

func (f *fakeClient) Close() error {
	f.closed = true
	return nil
}

func TestProducerEnqueuesTask(t *testing.T) {
	client := &fakeClient{}
	p := newProducer(client, Config{Queue: "it-queue"})

	if err := p.EnqueueBookingStatus(context.Background(), sampleEvent); err != nil {
		t.Fatalf("enqueue gagal: %v", err)
	}
	if len(client.tasks) != 1 || client.tasks[0].Type() != TaskTypeBookingStatus {
		t.Fatalf("task tidak masuk: %+v", client.tasks)
	}
	if got := optionValues(client.opts[0])[asynq.TaskIDOpt]; got != "booking-12-pending" {
		t.Fatalf("task id salah: %v", got)
	}
}

func TestProducerTreatsTaskIDConflictAsSuccess(t *testing.T) {
	client := &fakeClient{err: asynq.ErrTaskIDConflict}
	p := newProducer(client, Config{})

	if err := p.EnqueueBookingStatus(context.Background(), sampleEvent); err != nil {
		t.Fatalf("ErrTaskIDConflict harus dianggap sukses, dapat %v", err)
	}
}

func TestProducerReturnsOtherErrors(t *testing.T) {
	redisDown := errors.New("dial tcp 127.0.0.1:6379: connection refused")
	client := &fakeClient{err: redisDown}
	p := newProducer(client, Config{})

	err := p.EnqueueBookingStatus(context.Background(), sampleEvent)
	if !errors.Is(err, redisDown) {
		t.Fatalf("error Redis harus diteruskan, dapat %v", err)
	}
}

func TestProducerRejectsInvalidEventWithoutEnqueue(t *testing.T) {
	client := &fakeClient{}
	p := newProducer(client, Config{})
	bad := sampleEvent
	bad.Status = "created"

	if err := p.EnqueueBookingStatus(context.Background(), bad); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("harus ErrInvalidEvent, dapat %v", err)
	}
	if len(client.tasks) != 0 {
		t.Fatal("event tidak valid tidak boleh masuk antrean")
	}
}

func TestProducerCloseClosesClient(t *testing.T) {
	client := &fakeClient{}
	p := newProducer(client, Config{})

	if err := p.Close(); err != nil || !client.closed {
		t.Fatalf("Close harus menutup client: err=%v closed=%v", err, client.closed)
	}
}

func TestProducerSatisfiesEnqueuer(t *testing.T) {
	var _ interface {
		EnqueueBookingStatus(context.Context, model.BookingStatusEvent) error
	} = NewProducer(Config{})
}
