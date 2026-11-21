package queue

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
)

var sampleEvent = model.BookingStatusEvent{
	BookingID:  12,
	OwnerID:    3,
	Status:     model.StatusPending,
	OccurredAt: time.Date(2030, 12, 1, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600)),
}

func optionValues(opts []asynq.Option) map[asynq.OptionType]any {
	values := map[asynq.OptionType]any{}
	for _, opt := range opts {
		values[opt.Type()] = opt.Value()
	}
	return values
}

func TestTaskID(t *testing.T) {
	tests := []struct {
		bookingID int
		status    string
		want      string
	}{
		{12, model.StatusPending, "booking-12-pending"},
		{12, model.StatusApproved, "booking-12-approved"},
		{7, model.StatusCancelled, "booking-7-cancelled"},
	}
	for _, tt := range tests {
		if got := TaskID(tt.bookingID, tt.status); got != tt.want {
			t.Errorf("TaskID(%d,%s) = %s, want %s", tt.bookingID, tt.status, got, tt.want)
		}
	}
}

func TestNewBookingStatusTask(t *testing.T) {
	cfg := Config{Queue: "it-queue"}

	task, opts, err := newBookingStatusTask(sampleEvent, cfg)
	if err != nil {
		t.Fatalf("newBookingStatusTask: %v", err)
	}

	if task.Type() != TaskTypeBookingStatus || task.Type() != "notification:booking_status" {
		t.Fatalf("type task salah: %s", task.Type())
	}

	var payload model.BookingStatusEvent
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		t.Fatalf("payload bukan JSON event: %v", err)
	}
	if payload.BookingID != 12 || payload.OwnerID != 3 || payload.Status != model.StatusPending || !payload.OccurredAt.Equal(sampleEvent.OccurredAt) {
		t.Fatalf("payload salah: %+v", payload)
	}

	got := optionValues(opts)
	want := map[asynq.OptionType]any{
		asynq.TaskIDOpt:   "booking-12-pending",
		asynq.QueueOpt:    "it-queue",
		asynq.MaxRetryOpt: 5,
		asynq.TimeoutOpt:  30 * time.Second,
	}
	for optType, wantValue := range want {
		if got[optType] != wantValue {
			t.Errorf("opsi %v = %v, want %v", optType, got[optType], wantValue)
		}
	}
	if len(got) != len(want) {
		t.Errorf("opsi tak terduga: %v", got)
	}
}

func TestNewBookingStatusTaskDefaultsQueue(t *testing.T) {
	_, opts, err := newBookingStatusTask(sampleEvent, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got := optionValues(opts)[asynq.QueueOpt]; got != DefaultQueue {
		t.Fatalf("queue kosong harus memakai %s, dapat %v", DefaultQueue, got)
	}
}

func TestNewBookingStatusTaskRejectsInvalidEvent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.BookingStatusEvent)
	}{
		{"status tidak dikenal", func(e *model.BookingStatusEvent) { e.Status = "created" }},
		{"status kosong", func(e *model.BookingStatusEvent) { e.Status = "" }},
		{"booking id nol", func(e *model.BookingStatusEvent) { e.BookingID = 0 }},
		{"owner id nol", func(e *model.BookingStatusEvent) { e.OwnerID = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := sampleEvent
			tt.mutate(&e)
			if _, _, err := newBookingStatusTask(e, Config{}); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("harus ErrInvalidEvent, dapat %v", err)
			}
		})
	}
}

func TestBookingStatusText(t *testing.T) {
	tests := []struct {
		status      string
		wantTitle   string
		wantMessage string
	}{
		{model.StatusPending, "Peminjaman diajukan", "Peminjaman #12 berhasil diajukan dan menunggu persetujuan."},
		{model.StatusApproved, "Peminjaman disetujui", "Peminjaman #12 telah disetujui."},
		{model.StatusRejected, "Peminjaman ditolak", "Peminjaman #12 ditolak oleh pengelola."},
		{model.StatusCancelled, "Peminjaman dibatalkan", "Peminjaman #12 telah dibatalkan."},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			title, message, err := bookingStatusText(tt.status, 12)
			if err != nil {
				t.Fatalf("bookingStatusText: %v", err)
			}
			if title != tt.wantTitle || message != tt.wantMessage {
				t.Fatalf("teks salah: %q / %q", title, message)
			}
		})
	}
}

func TestBookingStatusTextUnknownStatus(t *testing.T) {
	for _, status := range []string{"", "created", "APPROVED"} {
		if _, _, err := bookingStatusText(status, 1); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("status %q harus ErrInvalidEvent, dapat %v", status, err)
		}
	}
}
