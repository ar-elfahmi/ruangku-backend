package main

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"ruangku-backend/app/service"
	"ruangku-backend/helper"
	"ruangku-backend/route"
)

// Test C-S1 (bagian 11.2 dokumen Phase 2): setiap route nyata di
// route.Register harus ada di apidocs/swagger.json dengan method yang sama,
// tanpa entri tambahan. Route dibaca dari aplikasi Fiber yang benar-benar
// didaftarkan; handler tidak pernah dipanggil sehingga service cukup berupa
// stub. route/route.go tidak diubah.
//
// Anotasi GET /health ditambahkan oleh A karena route/route.go milik A. Selama
// anotasi itu belum ada, endpoint tersebut dikecualikan dari perbandingan.
const (
	swaggerFile   = "apidocs/swagger.json"
	apiBasePath   = "/api/v1"
	totalAPIRoute = 24
	healthRoute   = "GET /health"
)

// Stub service: method value-nya cukup untuk registrasi route, dan akan panic
// bila benar-benar dipanggil (test ini tidak pernah mengirim request).
type stubAuth struct{ service.AuthService }

type stubSession struct{ service.SessionService }

type stubUser struct{ service.UserService }

type stubRoom struct{ service.RoomService }

type stubBooking struct{ service.BookingService }

type stubHistory struct{ service.HistoryService }

type stubNotification struct{ service.NotificationService }

func TestSwaggerCoversEveryRoute(t *testing.T) {
	actual := registeredRoutes(t)
	documented := swaggerRoutes(t)

	if got := len(actual); got != totalAPIRoute {
		t.Fatalf("route terdaftar = %d, want %d; perbarui test ini bila route memang bertambah", got, totalAPIRoute)
	}

	// /health milik A: dibandingkan hanya bila anotasinya sudah ada.
	delete(actual, healthRoute)
	delete(documented, healthRoute)

	missing, extra := diffRoutes(actual, documented)
	if len(missing) > 0 {
		t.Errorf("route belum dianotasi di %s:\n  %s", swaggerFile, strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("entri Swagger tidak punya route nyata:\n  %s", strings.Join(extra, "\n  "))
	}
	if len(missing) == 0 && len(extra) == 0 && len(documented) != totalAPIRoute-1 {
		t.Errorf("operasi Swagger = %d, want %d", len(documented), totalAPIRoute-1)
	}
}

// registeredRoutes mengembalikan himpunan "METHOD /path" dari aplikasi Fiber
// yang didaftarkan lewat route.Register, tanpa prefix /api/v1.
func registeredRoutes(t *testing.T) map[string]bool {
	t.Helper()

	app := fiber.New()
	route.Register(app, route.Dependencies{
		Pool:         (*pgxpool.Pool)(nil),
		JWT:          (*helper.JWTManager)(nil),
		Auth:         stubAuth{},
		Session:      stubSession{},
		User:         stubUser{},
		Room:         stubRoom{},
		Booking:      stubBooking{},
		History:      stubHistory{},
		Notification: stubNotification{},
	})

	routes := make(map[string]bool)
	for _, r := range app.GetRoutes(true) {
		if r.Method == fiber.MethodHead || r.Method == fiber.MethodOptions {
			continue
		}
		path := normalizeRoutePath(r.Path)
		if strings.HasPrefix(path, "/swagger") {
			continue
		}
		routes[r.Method+" "+path] = true
	}
	return routes
}

// swaggerRoutes membaca apidocs/swagger.json dan mengembalikan himpunan
// "METHOD /path" relatif terhadap basePath.
func swaggerRoutes(t *testing.T) map[string]bool {
	t.Helper()

	raw, err := os.ReadFile(swaggerFile)
	if err != nil {
		t.Fatalf("membaca %s: %v (jalankan perintah swag di bagian 3.6)", swaggerFile, err)
	}

	var doc struct {
		BasePath string                    `json:"basePath"`
		Paths    map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode %s: %v", swaggerFile, err)
	}
	if doc.BasePath != apiBasePath {
		t.Errorf("basePath Swagger = %q, want %q", doc.BasePath, apiBasePath)
	}

	routes := make(map[string]bool)
	for path, operations := range doc.Paths {
		for method := range operations {
			routes[strings.ToUpper(method)+" "+path] = true
		}
	}
	return routes
}

// normalizeRoutePath mengubah path Fiber menjadi bentuk Swagger: prefix
// /api/v1 dibuang, garis miring di akhir dihapus, dan ":param" menjadi
// "{param}".
func normalizeRoutePath(p string) string {
	p = strings.TrimPrefix(p, apiBasePath)
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}

	parts := strings.Split(p, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

// diffRoutes mengembalikan entri yang hanya ada di want (missing) dan yang
// hanya ada di got (extra), terurut agar pesan gagal stabil.
func diffRoutes(want, got map[string]bool) (missing, extra []string) {
	for route := range want {
		if !got[route] {
			missing = append(missing, route)
		}
	}
	for route := range got {
		if !want[route] {
			extra = append(extra, route)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
