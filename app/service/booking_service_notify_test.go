package service

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"ruangku-backend/app/model"
)

type fakeEnqueuer struct {
	err      error
	events   []model.BookingStatusEvent
	deadline time.Time
	hasLimit bool
	ctxErr   error
}

func (f *fakeEnqueuer) EnqueueBookingStatus(ctx context.Context, e model.BookingStatusEvent) error {
	f.events = append(f.events, e)
	f.deadline, f.hasLimit = ctx.Deadline()
	f.ctxErr = ctx.Err()
	return f.err
}

var notifyEvent = model.BookingStatusEvent{
	BookingID:  12,
	OwnerID:    3,
	Status:     model.StatusApproved,
	OccurredAt: time.Date(2030, 12, 1, 10, 0, 0, 0, time.UTC),
}

func TestNotifyBookingStatusSendsEventWithTimeLimit(t *testing.T) {
	f := &fakeEnqueuer{}

	notifyBookingStatus(f, notifyEvent)

	if len(f.events) != 1 || f.events[0] != notifyEvent {
		t.Fatalf("event tidak terkirim apa adanya: %+v", f.events)
	}
	if !f.hasLimit || time.Until(f.deadline) > notifyTimeout || f.ctxErr != nil {
		t.Fatalf("context harus baru dengan batas waktu <= %s (limit=%v err=%v)", notifyTimeout, f.hasLimit, f.ctxErr)
	}
}

func TestNotifyBookingStatusLogsFailureAndDoesNotPanic(t *testing.T) {
	var out bytes.Buffer
	original := log.Writer()
	log.SetOutput(&out)
	t.Cleanup(func() { log.SetOutput(original) })
	f := &fakeEnqueuer{err: errors.New("redis mati")}

	notifyBookingStatus(f, notifyEvent)

	logged := out.String()
	if !strings.Contains(logged, "redis mati") || !strings.Contains(logged, "12") || !strings.Contains(logged, model.StatusApproved) {
		t.Fatalf("kegagalan harus dicatat dengan booking dan status: %q", logged)
	}
}

func TestNotifyBookingStatusSkippedWithoutNotifier(t *testing.T) {
	notifyBookingStatus(nil, notifyEvent) // tidak boleh panic
}

func TestNewBookingServiceWorksWithoutOptions(t *testing.T) {
	svc, ok := NewBookingService(nil, nil, nil).(*bookingService)
	if !ok {
		t.Fatal("NewBookingService harus mengembalikan *bookingService")
	}
	if svc.notifier != nil {
		t.Fatal("tanpa WithNotifier, notifier harus nil (mode tanpa notifikasi)")
	}
}

func TestWithNotifierSetsNotifier(t *testing.T) {
	f := &fakeEnqueuer{}

	svc := NewBookingService(nil, nil, nil, WithNotifier(f)).(*bookingService)

	if svc.notifier != NotificationEnqueuer(f) {
		t.Fatal("WithNotifier harus memasang notifier")
	}
}
