package model

// BookingDetail adalah Booking dengan data ruang dan peminjam yang di-flatten.
type BookingDetail struct {
	Booking
	RoomCode string `json:"room_code"`
	RoomName string `json:"room_name"`
	UserName string `json:"user_name"`
}

// HistoryFilter: field nil atau string kosong berarti tidak difilter.
type HistoryFilter struct {
	UserID *int
	RoomID *int
	Status string
}
