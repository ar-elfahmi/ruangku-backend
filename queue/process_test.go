package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
)

type fakeStore struct {
	err      error
	inserted []model.Notification
}

func (f *fakeStore) InsertNotification(ctx context.Context, n model.Notification) error {
	f.inserted = append(f.inserted, n)
	return f.err
}

func taskFor(t *testing.T, e model.BookingStatusEvent) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return asynq.NewTask(TaskTypeBookingStatus, payload)
}

func TestProcessBookingStatusStoresNotification(t *testing.T) {
	store := &fakeStore{}
	p := &processor{store: store}

	if err := p.ProcessBookingStatus(context.Background(), taskFor(t, sampleEvent)); err != nil {
		t.Fatalf("ProcessBookingStatus: %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("harus menyimpan tepat satu notifikasi, dapat %d", len(store.inserted))
	}
	n := store.inserted[0]
	if n.UserID != 3 || n.BookingID != 12 || n.Status != model.StatusPending {
		t.Fatalf("pemilik/booking/status salah: %+v", n)
	}
	if n.Title != "Peminjaman diajukan" || n.Message != "Peminjaman #12 berhasil diajukan dan menunggu persetujuan." {
		t.Fatalf("teks salah: %q / %q", n.Title, n.Message)
	}
	if n.IsRead || n.ReadAt != nil || n.ID != "" {
		t.Fatalf("notifikasi baru harus belum dibaca dan tanpa id: %+v", n)
	}
	if !n.CreatedAt.Equal(sampleEvent.OccurredAt) {
		t.Fatalf("CreatedAt harus dari OccurredAt: %v", n.CreatedAt)
	}
}

func TestProcessBookingStatusDuplicateIsSuccess(t *testing.T) {
	for name, err := range map[string]error{
		"langsung":  repository.ErrDuplicate,
		"dibungkus": fmt.Errorf("insert: %w", repository.ErrDuplicate),
	} {
		t.Run(name, func(t *testing.T) {
			p := &processor{store: &fakeStore{err: err}}
			if got := p.ProcessBookingStatus(context.Background(), taskFor(t, sampleEvent)); got != nil {
				t.Fatalf("ErrDuplicate harus dianggap sukses, dapat %v", got)
			}
		})
	}
}

func TestProcessBookingStatusPermanentFailuresSkipRetry(t *testing.T) {
	badStatus := sampleEvent
	badStatus.Status = "created"
	zeroOwner := sampleEvent
	zeroOwner.OwnerID = 0

	tests := []struct {
		name string
		task *asynq.Task
	}{
		{"payload bukan JSON", asynq.NewTask(TaskTypeBookingStatus, []byte("{bukan json"))},
		{"payload kosong", asynq.NewTask(TaskTypeBookingStatus, nil)},
		{"status tidak dikenal", taskFor(t, badStatus)},
		{"owner id nol", taskFor(t, zeroOwner)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			p := &processor{store: store}

			err := p.ProcessBookingStatus(context.Background(), tt.task)

			if !errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("harus SkipRetry agar tidak diulang, dapat %v", err)
			}
			if len(store.inserted) != 0 {
				t.Fatal("task rusak tidak boleh menyimpan apa pun")
			}
		})
	}
}

func TestProcessBookingStatusStoreFailureIsRetried(t *testing.T) {
	mongoDown := errors.New("server selection timeout")
	p := &processor{store: &fakeStore{err: mongoDown}}

	err := p.ProcessBookingStatus(context.Background(), taskFor(t, sampleEvent))

	if !errors.Is(err, mongoDown) {
		t.Fatalf("error store harus diteruskan agar Asynq mengulang, dapat %v", err)
	}
	if errors.Is(err, asynq.SkipRetry) {
		t.Fatal("kegagalan sementara tidak boleh SkipRetry")
	}
}

func TestRepositorySatisfiesNotificationStore(t *testing.T) {
	var _ NotificationStore = repository.NotificationRepository(nil)
}
