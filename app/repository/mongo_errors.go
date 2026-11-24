package repository

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// TranslateMongoError mengubah error driver MongoDB menjadi error repository.
// Hanya duplicate key yang menjadi ErrDuplicate; error lain (misalnya gagal
// koneksi) dikembalikan apa adanya agar tidak menyamarkannya.
func TranslateMongoError(err error) error {
	if err == nil {
		return nil
	}
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicate
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}
