package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

const (
	messageRoomNotFound = "ruang tidak ditemukan"
	messageRoomCodeUsed = "kode ruang sudah digunakan"
)

type roomService struct {
	rooms repository.RoomRepository
}

func NewRoomService(r repository.RoomRepository) RoomService {
	return &roomService{rooms: r}
}

// List mengembalikan semua ruang aktif diurutkan berdasarkan kode.
//
//	@Summary		Daftar ruang aktif
//	@Tags			rooms
//	@Security		BearerAuth
//	@Success		200	{object}	model.WebResponse{data=[]model.Room}
//	@Failure		401	{object}	model.WebResponse
//	@Failure		500	{object}	model.WebResponse
//	@Router			/rooms [get]
func (s *roomService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	rooms, err := s.rooms.FindAllActive(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar ruang berhasil diambil", rooms)
}

// Get menampilkan satu ruang, termasuk yang tidak aktif.
//
//	@Summary		Detail ruang
//	@Tags			rooms
//	@Security		BearerAuth
//	@Param			id	path	int	true	"ID ruang"
//	@Success		200	{object}	model.WebResponse{data=model.Room}
//	@Failure		400	{object}	model.WebResponse
//	@Failure		401	{object}	model.WebResponse
//	@Failure		404	{object}	model.WebResponse
//	@Router			/rooms/{id} [get]
func (s *roomService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}
	room, err := s.rooms.FindByID(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(messageRoomNotFound)
	case err != nil:
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "ruang berhasil diambil", room)
}

// Create menambah ruang baru. Peran admin sudah dicek middleware di route;
// kode dinormalisasi agar "lab-01" dan "LAB-01" terdeteksi sebagai duplikat.
func (s *roomService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	room, err := s.rooms.Create(ctx, model.Room{
		Code:     NormalizeRoomCode(req.Code),
		Name:     req.Name,
		Building: req.Building,
		Capacity: req.Capacity,
	})
	switch {
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict(messageRoomCodeUsed)
	case err != nil:
		return helper.Internal(err)
	}
	return helper.Created(c, "ruang berhasil dibuat", room, "/api/v1/rooms/"+strconv.Itoa(room.ID))
}

// Update mengganti seluruh field ruang (PUT: semua field wajib diisi).
func (s *roomService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}
	var req model.RoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	room, err := s.rooms.Update(ctx, model.Room{
		ID:       id,
		Code:     NormalizeRoomCode(req.Code),
		Name:     req.Name,
		Building: req.Building,
		Capacity: req.Capacity,
	})
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(messageRoomNotFound)
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict(messageRoomCodeUsed)
	case err != nil:
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "ruang berhasil diperbarui", room)
}

// Delete menonaktifkan ruang (soft delete) agar riwayat booking tetap terbaca.
func (s *roomService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}
	err := s.rooms.SoftDelete(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(messageRoomNotFound)
	case err != nil:
		return helper.Internal(err)
	}
	return helper.NoContent(c)
}

// Availability menampilkan slot pending/approved pada sebuah ruang untuk satu
// tanggal, diinterpretasikan pada zona waktu aplikasi (Asia/Jakarta).
//
//	@Summary		Ketersediaan ruang pada satu tanggal
//	@Tags			rooms
//	@Security		BearerAuth
//	@Param			id		path	int		true	"ID ruang"
//	@Param			date	query	string	true	"Tanggal dengan format YYYY-MM-DD"
//	@Success		200		{object}	model.WebResponse{data=model.Availability}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Failure		404		{object}	model.WebResponse
//	@Router			/rooms/{id}/availability [get]
func (s *roomService) Availability(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}
	date := c.Query("date")
	dayStart, dayEnd, err := ParseDay(date, time.Local)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	room, err := s.rooms.FindByID(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(messageRoomNotFound)
	case err != nil:
		return helper.Internal(err)
	}

	booked, err := s.rooms.FindBookedSlots(ctx, id, dayStart, dayEnd)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "ketersediaan ruang berhasil diambil", model.Availability{
		Room:   room,
		Date:   date,
		Booked: booked,
	})
}

var _ RoomService = (*roomService)(nil)
