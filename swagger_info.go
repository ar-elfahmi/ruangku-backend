// Package main berisi entry point API RuangKu.
//
// Informasi umum Swagger di bawah ini dibaca oleh `swag init` (bagian 3.6
// dokumen Phase 2) dan tidak memengaruhi runtime.
//
//	@title						RuangKu Backend API
//	@version					1.0
//	@description				API peminjaman ruang kampus: autentikasi, ruang, peminjaman, riwayat, notifikasi, dan refresh token. Semua endpoint berada di bawah /api/v1.
//	@BasePath					/api/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Access token dengan format: Bearer <token>
package main
