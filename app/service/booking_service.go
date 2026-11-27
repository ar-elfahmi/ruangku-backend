package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

const (
	messageBookingNotFound = "peminjaman tidak ditemukan"
	messageStatusConflict  = "status peminjaman tidak dapat diubah"
	messageOverlap         = "ruang sudah dipesan pada jam tersebut"
	messageForbidden       = "akses ditolak"
	messageInvalidToken    = "token tidak valid atau sudah kedaluwarsa"
)

// notifyTimeout membatasi waktu mengirim event ke antrean. Context-nya baru,
// bukan context request, agar pengiriman tidak ikut batal bila klien putus.
const notifyTimeout = 2 * time.Second

type bookingService struct {
	bookings repository.BookingRepository
	rooms    repository.RoomRepository
	tx       *repository.TxManager
	notifier NotificationEnqueuer // nil = mode tanpa notifikasi
}

// BookingOption mengubah konfigurasi bookingService. Dipakai agar signature
// NewBookingService tetap kompatibel saat fitur baru belum di-wire.
type BookingOption func(*bookingService)

// WithNotifier mengaktifkan pengiriman event notifikasi setelah booking dibuat
// atau berubah status. Tanpa opsi ini, hook dilewati.
func WithNotifier(n NotificationEnqueuer) BookingOption {
	return func(s *bookingService) { s.notifier = n }
}

func NewBookingService(b repository.BookingRepository, r repository.RoomRepository, tx *repository.TxManager, opts ...BookingOption) BookingService {
	s := &bookingService{bookings: b, rooms: r, tx: tx}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// notifyBookingStatus mengirim event ke antrean. Kegagalan hanya dicatat dan
// tidak mengubah response HTTP: perubahan di PostgreSQL sudah commit dan
// itulah sumber kebenarannya.
func notifyBookingStatus(n NotificationEnqueuer, e model.BookingStatusEvent) {
	if n == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	if err := n.EnqueueBookingStatus(ctx, e); err != nil {
		log.Printf("notifikasi: gagal mengirim event booking %d (%s): %v", e.BookingID, e.Status, err)
	}
}

// Create mengajukan peminjaman baru untuk peminjam yang sedang login.
//
//	@Summary		Ajukan peminjaman
//	@Tags			bookings
//	@Security		BearerAuth
//	@Param			request	body	model.CreateBookingRequest	true	"Data peminjaman"
//	@Success		201		{object}	model.WebResponse{data=model.Booking}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Failure		403		{object}	model.WebResponse
//	@Failure		404		{object}	model.WebResponse
//	@Failure		409		{object}	model.WebResponse
//	@Failure		422		{object}	model.WebResponse
//	@Router			/bookings [post]
func (s *bookingService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateBookingRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if err := ValidateBookingWindow(req.StartTime, req.EndTime, time.Now()); err != nil {
		return helper.BadRequest(err.Error())
	}

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	room, err := s.rooms.FindByID(ctx, req.RoomID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("ruang tidak ditemukan")
	case err != nil:
		return helper.Internal(err)
	case !room.IsActive:
		return helper.BadRequest("ruang tidak aktif")
	}

	var created model.Booking
	err = s.tx.Do(ctx, func(tx repository.DBTX) error {
		repo := s.bookings.WithTx(tx)
		var err error
		created, err = repo.Create(ctx, model.Booking{
			UserID:    user.ID,
			RoomID:    req.RoomID,
			Purpose:   req.Purpose,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		})
		if err != nil {
			return err
		}
		return repo.InsertLog(ctx, model.BookingLog{
			BookingID: created.ID,
			ActorID:   user.ID,
			ToStatus:  model.StatusPending,
		})
	})
	switch {
	case errors.Is(err, repository.ErrOverlap):
		return helper.Conflict(messageOverlap)
	case err != nil:
		return helper.Internal(err)
	}
	notifyBookingStatus(s.notifier, model.BookingStatusEvent{
		BookingID:  created.ID,
		OwnerID:    created.UserID,
		Status:     model.StatusPending,
		OccurredAt: time.Now(),
	})
	return helper.Created(c, "peminjaman berhasil diajukan", created, "")
}

// Cancel membatalkan peminjaman milik peminjam.
//
//	@Summary		Batalkan peminjaman
//	@Tags			bookings
//	@Security		BearerAuth
//	@Param			id		path	int							true	"ID peminjaman"
//	@Param			request	body	model.StatusChangeRequest	false	"Catatan pembatalan (opsional)"
//	@Success		200		{object}	model.WebResponse{data=model.Booking}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Failure		403		{object}	model.WebResponse
//	@Failure		404		{object}	model.WebResponse
//	@Failure		409		{object}	model.WebResponse
//	@Failure		422		{object}	model.WebResponse
//	@Router			/bookings/{id}/cancel [patch]
func (s *bookingService) Cancel(c *fiber.Ctx) error {
	return s.changeStatus(c, model.StatusCancelled, "peminjaman berhasil dibatalkan")
}

// Approve menyetujui peminjaman; hanya pengelola.
//
//	@Summary		Setujui peminjaman
//	@Tags			bookings
//	@Security		BearerAuth
//	@Param			id		path	int							true	"ID peminjaman"
//	@Param			request	body	model.StatusChangeRequest	false	"Catatan persetujuan (opsional)"
//	@Success		200		{object}	model.WebResponse{data=model.Booking}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Failure		403		{object}	model.WebResponse
//	@Failure		404		{object}	model.WebResponse
//	@Failure		409		{object}	model.WebResponse
//	@Failure		422		{object}	model.WebResponse
//	@Router			/bookings/{id}/approve [patch]
func (s *bookingService) Approve(c *fiber.Ctx) error {
	return s.changeStatus(c, model.StatusApproved, "peminjaman berhasil disetujui")
}

func (s *bookingService) Reject(c *fiber.Ctx) error {
	return s.changeStatus(c, model.StatusRejected, "peminjaman berhasil ditolak")
}

// changeStatus dipakai bersama oleh Cancel, Approve, dan Reject karena
// alurnya identik; hanya status tujuan dan pesan sukses yang berbeda.
func (s *bookingService) changeStatus(c *fiber.Ctx, target, successMessage string) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}

	// Body opsional; BodyParser gagal pada body kosong.
	var req model.StatusChangeRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("body request tidak valid")
		}
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	booking, err := s.bookings.FindByID(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(messageBookingNotFound)
	case err != nil:
		return helper.Internal(err)
	}

	// 403 diperiksa sebelum 409 agar peran/kepemilikan yang salah tidak
	// membocorkan status booking milik orang lain.
	isOwner := booking.UserID == user.ID
	if !CanActOn(target, user.Role, isOwner) {
		return helper.Forbidden(messageForbidden)
	}
	if !CanTransition(booking.Status, target, user.Role, isOwner) {
		return helper.Conflict(messageStatusConflict)
	}

	var updated model.Booking
	err = s.tx.Do(ctx, func(tx repository.DBTX) error {
		repo := s.bookings.WithTx(tx)
		var err error
		updated, err = repo.UpdateStatus(ctx, id, booking.Status, target)
		if err != nil {
			return err
		}
		from := booking.Status
		return repo.InsertLog(ctx, model.BookingLog{
			BookingID:  id,
			ActorID:    user.ID,
			FromStatus: &from,
			ToStatus:   target,
			Note:       req.Note,
		})
	})
	switch {
	case errors.Is(err, repository.ErrConflict):
		return helper.Conflict(messageStatusConflict)
	case err != nil:
		return helper.Internal(err)
	}
	notifyBookingStatus(s.notifier, model.BookingStatusEvent{
		BookingID:  updated.ID,
		OwnerID:    updated.UserID,
		Status:     target,
		OccurredAt: time.Now(),
	})
	return helper.Success(c, fiber.StatusOK, successMessage, updated)
}

func (s *bookingService) Logs(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}
	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	booking, err := s.bookings.FindByID(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(messageBookingNotFound)
	case err != nil:
		return helper.Internal(err)
	}
	if booking.UserID != user.ID && user.Role != model.RolePengelola {
		return helper.Forbidden(messageForbidden)
	}

	logs, err := s.bookings.FindLogs(ctx, id)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "riwayat status peminjaman berhasil diambil", logs)
}

var _ BookingService = (*bookingService)(nil)
