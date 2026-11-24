package service

import (
	"regexp"

	"ruangku-backend/helper"
)

const (
	// notificationListLimit sama dengan batas daftar di repository.
	notificationListLimit = 100

	messageNotificationNotFound  = "notifikasi tidak ditemukan"
	messageNotificationIDInvalid = "id tidak valid"
	messageReadFilterInvalid     = "is_read tidak valid"

	messageNotificationList    = "daftar notifikasi berhasil diambil"
	messageNotificationUnread  = "jumlah notifikasi belum dibaca berhasil diambil"
	messageNotificationRead    = "notifikasi berhasil ditandai sudah dibaca"
	messageNotificationReadAll = "semua notifikasi berhasil ditandai sudah dibaca"
)

var notificationIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)

// ParseUnreadFilter membaca query is_read. Kontrak API hanya mendukung
// ?is_read=false (hanya yang belum dibaca); kosong berarti semua.
func ParseUnreadFilter(raw string) (bool, error) {
	switch raw {
	case "":
		return false, nil
	case "false":
		return true, nil
	default:
		return false, helper.BadRequest(messageReadFilterInvalid)
	}
}

// IsNotificationID memastikan id berbentuk ObjectID hex 24 karakter.
func IsNotificationID(id string) bool {
	return notificationIDPattern.MatchString(id)
}
