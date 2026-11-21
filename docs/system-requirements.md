# RuangKu Backend — System Requirements & Conventions

This document is the **single source of truth** for how the code, API, database, and Git history look. When it conflicts with the implementation plan, this document wins. When it doesn't cover something, ask the Git Master (Member A) and add the answer here in a `docs(conventions): …` PR.

Every PR is reviewed against this document.

---

## 1. Runtime & Environment

| Item | Requirement |
|---|---|
| Go | 1.26, module `ruangku-backend` |
| Web framework | Fiber v2 |
| Database | PostgreSQL 16 via `docker compose`, driver pgx v5 (`pgxpool`) |
| Time zone | `Asia/Jakarta` (+07:00) for all API input and output |
| Port | `APP_PORT`, default `8080` |
| Base path | `/api/v1` |
| Document database | MongoDB 7 via `docker compose`, official Go driver (notifications) |
| Queue | Redis 7 via `docker compose`, asynq (background jobs) |
| API docs | Swagger via swaggo, served at `/swagger/index.html` (added in Wave 4) |
| Processes | Two: the API (`go run .`) and the notification worker (`go run ./cmd/worker`) |

### Environment variables

`UPPER_SNAKE_CASE`, read **only** through `config.GetEnv` / `config.GetEnvInt`. Never call `os.Getenv` anywhere else.

| Key | Example | Used by |
|---|---|---|
| `APP_PORT` | `8080` | main |
| `APP_TIMEZONE` | `Asia/Jakarta` | main, `cmd/worker` (set `time.Local`) |
| `DB_URL` | `postgres://ruangku:ruangku@localhost:5432/ruangku?sslmode=disable` | database (API only) |
| `JWT_SECRET` | ≥ 32 characters | helper/jwt (API only) |
| `JWT_TTL_MINUTES` | `15` (access token lifetime) | helper/jwt (API only) |
| `JWT_REFRESH_SECRET` | ≥ 32 characters, **must differ from `JWT_SECRET`** | helper/jwt (API only) |
| `REFRESH_TTL_HOURS` | `168` (refresh token lifetime) | helper/jwt (API only) |
| `ADMIN_NAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` | `Admin` / `admin@ruangku.local` / `admin12345` | `AuthService.EnsureAdmin` (API only) |
| `MONGO_URI` | `mongodb://localhost:27017` | database (API and worker) |
| `MONGO_DB` | `ruangku` | database (API and worker) |
| `REDIS_ADDR` | `localhost:6379` | config (API and worker) |
| `WORKER_CONCURRENCY` | `5` (jobs processed at the same time) | `cmd/worker` only |

Any new key must be added to `.env.example` and to this table in the same PR. `.env` is never committed.

The worker process reads only `APP_TIMEZONE`, `MONGO_URI`, `MONGO_DB`, `REDIS_ADDR` and `WORKER_CONCURRENCY`. It never reads `DB_URL`, `JWT_*` or `ADMIN_*`, so it can run without PostgreSQL.

`main.go` and `cmd/worker/main.go` import `_ "time/tzdata"` and set `time.Local` from `APP_TIMEZONE`. This makes the time zone load on Windows machines without tzdata, and makes pgx return `TIMESTAMPTZ` values in +07:00.

---

## 2. Language

| Where | Language | Example |
|---|---|---|
| Identifiers (packages, types, funcs, vars, JSON keys, DB names, URLs) | **English** | `FindAllActive`, `start_time` |
| `message` and validation messages in API responses | **Indonesian**, lowercase first letter, no trailing period | `"ruang berhasil dibuat"` |
| Code comments | Indonesian or English, but one language per file | |
| Commit messages, PR titles, branch names | **English** | `feat(repository): add query to get all rooms` |
| README | Indonesian | |

---

## 3. Project Layout & Layer Rules

```
main.go  route/  middleware/  helper/  config/  database/  migrations/
app/model/  app/repository/  app/service/  queue/  cmd/worker/  http/  docs/
```

| Layer | May import | Must NOT | Responsibility |
|---|---|---|---|
| `app/model` | stdlib only | `fiber`, `pgx`, other app packages | Structs, constants, request/response DTOs, `validate` tags |
| `app/repository` | `model`, `pgx`, `mongo-driver` | `fiber`, `helper`, `service`, `queue` | SQL (PostgreSQL) and MongoDB queries only. Returns `model` types and repository errors |
| `app/service` (`*_rules.go`) | stdlib, `model` | `fiber`, `repository`, `helper`, DB, clock | Pure functions. Time is passed in as `now time.Time` |
| `app/service` (`*_service.go`) | `model`, `repository`, `helper`, `fiber` | SQL, `pgx`, `mongo-driver`, `queue`, `asynq` | Parse → validate → rules → repository → map errors → respond. A service reaches the queue only through the `NotificationEnqueuer` interface in `app/service/contracts.go`; `main.go` injects the implementation |
| `middleware` | `helper`, `model`, `fiber` | `repository`, `service` | Auth and role checks only |
| `route` | `service` interfaces, `middleware`, `fiber` | business logic | Route table only |
| `helper` | `model`, `fiber`, libs | `repository`, `service` | Shared response, error, validation, JWT, password, and context helpers |
| `queue` | stdlib, `model`, `repository` (interfaces only), `asynq` | `fiber`, `pgx`, `mongo-driver`, `service` | Producer (`EnqueueNotification`), worker (`Process…` methods, `Start`, `Shutdown`), task payload, and the pure rules `NotificationTaskID` and `BuildNotification` |
| `cmd/worker` | `config`, `database`, `repository`, `queue` | `fiber`, `route`, `service`, PostgreSQL | Worker process entry point. Wiring only, like `main.go` |

**Forbidden everywhere:** the words `controller` and `handler` in any folder, file, type, function, or variable name (`fiber.Handler` inside `middleware/` is the only exception). asynq's own `Handler`/`HandleFunc` names may be *called* inside `queue/`, but never appear in our own identifiers: our types are `Producer` and `Worker`, and our methods start with `Process`.

---

## 4. Naming Conventions

### 4.1 Go

| Thing | Rule | Example |
|---|---|---|
| Package | lowercase, single word, singular | `service`, `repository` |
| File | `snake_case`, `<feature>_<layer>.go` | `room_repository.go`, `booking_rules.go`, `booking_rules_test.go` |
| Interface | `PascalCase` `<Feature><Layer>` | `RoomRepository`, `BookingService` |
| Implementing struct | **unexported** camelCase, same name | `roomRepository`, `bookingService` |
| Constructor | `New<Interface>` returning the **interface** | `func NewRoomService(...) RoomService` |
| Compile-time check | At the bottom of every implementation file | `var _ RoomService = (*roomService)(nil)` |
| Receiver | One letter, consistent within a type | `func (s *roomService)`, `func (r *roomRepository)` |
| Sentinel error | `Err<Reason>` declared with `errors.New`, message Indonesian | `ErrOverlap = errors.New("jadwal bentrok")` |
| Constant groups | `<Kind><Value>` | `RoleAdmin`, `StatusPending` |
| Initialisms | Fully upper case | `ID`, `URL`, `JWT`, `HTTP` (`UserID`, not `UserId`) |
| Local variables | Short but meaningful. No `data`, `temp`, `obj`, `res2` | `room`, `req`, `booking` |

### 4.2 Repository method verbs

| Verb | Meaning | Returns on no row |
|---|---|---|
| `FindByID`, `FindByEmail`, `Find…` | Read one or many | one: `ErrNotFound`; many: empty slice `[]T{}` (never `nil`) |
| `Create` | INSERT … RETURNING | — |
| `Update` / `Update<Field>` | UPDATE … RETURNING | `ErrNotFound` (or `ErrConflict` for guarded updates) |
| `SoftDelete` | Set `is_active=false` | `ErrNotFound` |
| `Insert<Thing>` | INSERT with no return value (logs, notifications) | — |
| `Consume` | Guarded single-use `UPDATE … RETURNING` (refresh token rotation) | `ErrConflict` |
| `Revoke<Scope>` | `UPDATE` that revokes, with no return value (`RevokeFamily`) | — |
| `Mark<State>` | Change a flag. `MarkRead` returns the updated item; `MarkAllRead` returns the count | `ErrNotFound` when the item is not the caller's |

### 4.3 Service method names

Service methods are named after the action the user performs: `List`, `Get`, `Create`, `Update`, `Delete`, `Approve`, `Cancel`, `Mine`. The service method is the route's endpoint, so there is no separate wrapper function.

### 4.4 Rule function names

Rule functions are verb-first and pure: `NormalizeEmail`, `NormalizeRoomCode`, `ValidateBookingWindow`, `CanTransition`, `ParseDay`, `ParseStatusFilter`. Boolean rules start with `Can`/`Is`/`Has`.

### 4.5 Database

| Thing | Rule | Example |
|---|---|---|
| Table | `snake_case`, **plural** | `rooms`, `booking_logs` |
| Column | `snake_case` | `start_time`, `password_hash` |
| Primary key | `id SERIAL` | |
| Foreign key column | `<singular>_id` | `room_id`, `actor_id` |
| Boolean | `is_` / `has_` prefix | `is_active` |
| Timestamps | `TIMESTAMPTZ`, `created_at` default `now()` | |
| Enum-like values | `VARCHAR` + `CHECK (col IN (...))`, lowercase values | `status IN ('pending',…)` |
| Constraint/index names | `<table>_<column(s)>_<suffix>`; suffix `pkey`, `key` (unique), `fkey`, `check`, `excl`, `idx` | `users_email_lower_key`, `bookings_room_time_excl` |
| SQL keywords | UPPER CASE | `SELECT id, code FROM rooms WHERE …` |

### 4.6 HTTP

| Thing | Rule | Example |
|---|---|---|
| Path segments | lowercase, plural nouns, `kebab-case` if multi-word, no trailing slash, no verbs except for state actions | `/rooms/:id`, `/bookings/:id/approve` |
| Path params | `:id` (camelCase only if a second param is ever needed) | `/rooms/:id/availability` |
| Query params | `snake_case` | `?status=approved&room_id=3`, `?date=2026-12-01` |
| JSON keys | `snake_case`, in both request and response | `start_time`, `access_token` |

### 4.7 Git

| Thing | Rule | Example |
|---|---|---|
| Branch | `<type>/<short-kebab-scope>` | `feat/rooms`, `feat/booking-rules`, `fix/room-duplicate-code`, `docs/readme-design` |
| Commit | Conventional Commits: `<type>(<scope>): <imperative, lowercase summary>` with no period, ≤ 72 chars | `feat(service): reject booking outside time window` |
| Types | `feat`, `fix`, `refactor`, `test`, `docs`, `chore` | |
| Scopes | `model`, `repository`, `service`, `route`, `middleware`, `helper`, `config`, `database`, `migration`, `api` (for `.http` files), `readme`, `contract` | |
| PR title | Same format as a commit | `feat(rooms): implement room management` |

---

## 5. HTTP API Rules

### 5.1 Request rules

- Bodies are JSON. Send `Content-Type: application/json`.
- Unknown JSON fields are **ignored**. Fields the server owns (`id`, `role` on register, `status`, `user_id`, `created_at`, `is_active`) are never read from the body.
- A malformed JSON body returns `400 "body request tidak valid"`.
- Strings are trimmed by the rule functions before they are saved (`NormalizeEmail`, `NormalizeRoomCode`).
- A protected endpoint requires `Authorization: Bearer <access_token>`.

### 5.2 Method semantics

| Method | Used for | Success status | Body |
|---|---|---|---|
| GET | read | 200 | envelope |
| POST | create | 201 (+ `Location` header when a GET-by-id route exists) | envelope with the created object |
| PUT | full replace (all fields required) | 200 | envelope with the updated object |
| PATCH | state action or single-field change | 200 | envelope with the updated object |
| DELETE | delete (soft) | 204 | **no body** |

A `POST` that performs an **action** instead of creating a resource returns 200 (`/auth/login`, `/auth/refresh`), or 204 with no body when there is nothing to return (`/auth/logout`).

### 5.3 Status codes (the only ones allowed)

| Code | When | `message` |
|---|---|---|
| 200 | OK | per endpoint |
| 201 | Created | per endpoint |
| 204 | Deleted | — |
| 400 | Malformed JSON, bad path id, bad query param, a business rule broken by input (e.g. time window) | see §6 |
| 401 | Missing/invalid/expired token, wrong login | `"token tidak valid atau sudah kedaluwarsa"` / `"email atau password salah"` |
| 403 | Authenticated but the role or ownership is wrong | `"akses ditolak"` |
| 404 | Resource doesn't exist | `"<resource> tidak ditemukan"` |
| 409 | Duplicate, overlap, or invalid state transition | see §6 |
| 422 | Struct validation failed (`validate` tags) | `"data tidak valid"` + `errors` |
| 500 | Unexpected error (details only in the server log) | `"terjadi kesalahan pada server"` |
| 501 | Skeleton not yet implemented | `"fitur belum tersedia"` |
| 503 | `/health` when the DB is unreachable | `"database tidak dapat dihubungi"` |

**400 vs 422:** 422 is only for `validate` tag failures on the request struct. Everything else the client did wrong is 400.

### 5.4 Response envelope

Every response except 204 uses **exactly** this shape (`model.WebResponse`), with keys in this order:

```jsonc
{
  "success": true,          // always present
  "message": "…",           // always present, Indonesian
  "data":    { } | [ ],     // present on success; omitted on error
  "meta":    { },           // reserved for pagination; omitted for now
  "errors":  { }            // present only on 422
}
```

Success with an object:
```json
{ "success": true, "message": "ruang berhasil diambil", "data": { "id": 1, "code": "LAB-01", "...": "..." } }
```

Success with a list. An empty list is `[]`, **never** `null`:
```json
{ "success": true, "message": "daftar ruang berhasil diambil", "data": [] }
```

Error:
```json
{ "success": false, "message": "ruang tidak ditemukan" }
```

Validation error (422). The keys in `errors` are the **JSON field names**:
```json
{
  "success": false,
  "message": "data tidak valid",
  "errors": { "email": "format email tidak valid", "password": "minimal 8 karakter" }
}
```

Services never call `c.JSON` directly. They use `helper.Success`, `helper.Created`, `helper.NoContent`, or `return helper.<Error>(...)`, and the error handler in `config.NewApp` writes the error envelope.

### 5.5 Data formats

| Type | Format | Example |
|---|---|---|
| ID | positive integer | `12` |
| Notification ID | 24-character hex string (MongoDB ObjectID), the only non-integer ID | `"6560f0c2a1b2c3d4e5f60718"` |
| Timestamp (in and out) | RFC 3339 **with offset**. Output is always `+07:00` | `"2026-12-01T10:00:00+07:00"` |
| Date (query) | `YYYY-MM-DD`, interpreted in Asia/Jakarta | `2026-12-01` |
| Enum | lowercase string | `"pending"`, `"peminjam"` |
| Nullable | JSON `null`, never `""` | `"from_status": null` |
| Boolean | JSON boolean | `"is_active": true` |
| Secrets | never serialized (`json:"-"`) | `password_hash` |

A timestamp without an offset (`"2026-12-01T10:00:00"`) fails JSON parsing and returns 400 `"body request tidak valid"`.

### 5.6 Pagination

There is no pagination in this version, so lists return everything. If it's added later, it uses `?page=1&limit=20` and `"meta": {"page":1,"limit":20,"total":57,"total_pages":3}`. That requires a `refactor(contract)` PR first.

---

## 6. Messages Catalogue

Use these exact strings. New strings follow the same pattern and are added here in the PR that introduces them.

### 6.1 Success messages: `<resource> berhasil <past-tense verb>`

| Endpoint | `message` |
|---|---|
| GET /health | `"service berjalan normal"` |
| POST /auth/register | `"registrasi berhasil"` |
| POST /auth/login | `"login berhasil"` |
| POST /auth/refresh | `"token berhasil diperbarui"` |
| POST /auth/logout | *(204, no body)* |
| GET /auth/me | `"profil berhasil diambil"` |
| PATCH /users/:id/role | `"role user berhasil diubah"` |
| GET /rooms | `"daftar ruang berhasil diambil"` |
| GET /rooms/:id | `"ruang berhasil diambil"` |
| POST /rooms | `"ruang berhasil dibuat"` |
| PUT /rooms/:id | `"ruang berhasil diperbarui"` |
| GET /rooms/:id/availability | `"ketersediaan ruang berhasil diambil"` |
| POST /bookings | `"peminjaman berhasil diajukan"` |
| PATCH /bookings/:id/approve | `"peminjaman berhasil disetujui"` |
| PATCH /bookings/:id/reject | `"peminjaman berhasil ditolak"` |
| PATCH /bookings/:id/cancel | `"peminjaman berhasil dibatalkan"` |
| GET /bookings/:id/logs | `"riwayat status peminjaman berhasil diambil"` |
| GET /bookings/mine | `"riwayat peminjaman berhasil diambil"` |
| GET /bookings | `"daftar peminjaman berhasil diambil"` |
| GET /notifications | `"notifikasi berhasil diambil"` |
| PATCH /notifications/:id/read | `"notifikasi berhasil ditandai dibaca"` |
| PATCH /notifications/read-all | `"semua notifikasi berhasil ditandai dibaca"` |

### 6.2 Error messages

| Situation | Status | `message` |
|---|---|---|
| Malformed body | 400 | `body request tidak valid` |
| Bad path id | 400 | `id tidak valid` |
| Bad `date` query | 400 | `format tanggal harus YYYY-MM-DD` |
| Bad `status` query | 400 | `status tidak dikenal` |
| Bad `room_id` query | 400 | `room_id tidak valid` |
| Bad `unread` query | 400 | `unread tidak valid` |
| start ≥ end | 400 | `waktu selesai harus setelah waktu mulai` |
| start in the past | 400 | `waktu mulai harus di masa depan` |
| Room inactive (on booking) | 400 | `ruang tidak aktif` |
| Wrong login | 401 | `email atau password salah` |
| Missing/invalid token | 401 | `token tidak valid atau sudah kedaluwarsa` |
| Invalid refresh token (bad signature, expired, wrong type, already used, revoked) | 401 | `refresh token tidak valid atau sudah kedaluwarsa` |
| Role/ownership | 403 | `akses ditolak` |
| Not found | 404 | `user tidak ditemukan` / `ruang tidak ditemukan` / `peminjaman tidak ditemukan` / `notifikasi tidak ditemukan` |
| Duplicate email | 409 | `email sudah terdaftar` |
| Duplicate room code | 409 | `kode ruang sudah digunakan` |
| Overlap | 409 | `ruang sudah dipesan pada jam tersebut` |
| Transition not allowed / changed concurrently | 409 | `status peminjaman tidak dapat diubah` |
| Validation | 422 | `data tidak valid` |
| Unexpected | 500 | `terjadi kesalahan pada server` |
| Skeleton | 501 | `fitur belum tersedia` |

### 6.3 Validation messages (`helper.ValidateStruct`)

| Tag | Message |
|---|---|
| `required` | `wajib diisi` |
| `email` | `format email tidak valid` |
| `min` (string) | `minimal {param} karakter` |
| `max` (string) | `maksimal {param} karakter` |
| `min` (number) | `minimal {param}` |
| `max` (number) | `maksimal {param}` |
| `oneof` | `harus salah satu dari: {param}` |

---

## 7. Resource Shapes

These are the JSON representations of the `model` structs. Field order in responses follows this order.

**User**
```json
{ "id": 3, "name": "Citra", "email": "citra@kampus.ac.id", "role": "peminjam", "created_at": "2026-11-20T09:00:00+07:00" }
```

**Token**
```json
{ "access_token": "eyJhbGciOi...", "refresh_token": "eyJhbGciOi...", "expires_in": 900 }
```
`expires_in` is the lifetime of the access token in seconds.

**Room**
```json
{ "id": 1, "code": "LAB-01", "name": "Lab Komputer 1", "building": "Gedung C", "capacity": 40, "is_active": true, "created_at": "2026-11-20T09:00:00+07:00" }
```

**Booking**
```json
{ "id": 7, "user_id": 3, "room_id": 1, "purpose": "Rapat BEM", "start_time": "2026-12-01T10:00:00+07:00", "end_time": "2026-12-01T12:00:00+07:00", "status": "pending", "created_at": "2026-11-20T09:05:00+07:00" }
```

**BookingLog**
```json
{ "id": 15, "booking_id": 7, "actor_id": 2, "from_status": "pending", "to_status": "approved", "note": "", "created_at": "2026-11-21T08:00:00+07:00" }
```

**BookingDetail**: Booking fields, flattened, plus:
```json
{ "…booking fields…": "", "room_code": "LAB-01", "room_name": "Lab Komputer 1", "user_name": "Citra" }
```

**Availability**
```json
{
  "room": { "…Room…": "" },
  "date": "2026-12-01",
  "booked": [
    { "booking_id": 7, "start_time": "2026-12-01T10:00:00+07:00", "end_time": "2026-12-01T12:00:00+07:00", "status": "approved" }
  ]
}
```

**Notification**
```json
{ "id": "6560f0c2a1b2c3d4e5f60718", "user_id": 3, "booking_id": 7, "status": "approved", "title": "Peminjaman disetujui", "message": "…", "is_read": false, "created_at": "2026-11-21T08:00:00+07:00" }
```
`status` is the booking status that triggered the notification.

**Read-all result** (`data` of `PATCH /notifications/read-all`)
```json
{ "updated": 2 }
```

---

## 8. Endpoint Catalogue

Legend for **Auth**: `—` public · `any` any logged-in user · otherwise the required role.

### 8.1 Health & Auth & Users

| Method & Path | Auth | Request body | Success `data` | Errors |
|---|---|---|---|---|
| `GET /health` | — | — | `{ "database": "up" }` | 503 |
| `POST /auth/register` | — | `{ "name", "email", "password" }` | User (role always `peminjam`) | 400, 409, 422 |
| `POST /auth/login` | — | `{ "email", "password" }` | Token (includes `refresh_token`) | 400, 401, 422 |
| `POST /auth/refresh` | — | `{ "refresh_token" }` | Token (new access and refresh token) | 400, 401, 422 |
| `POST /auth/logout` | any | `{ "refresh_token" }` | *(204, no body)* | 400, 401, 422 |
| `GET /auth/me` | any | — | User | 401 |
| `PATCH /users/:id/role` | admin | `{ "role": "pengelola" }` | User | 400, 401, 403, 404, 422 |

Refresh rules: a refresh token works **once**. `POST /auth/refresh` returns a new access token and a new refresh token, and the old refresh token stops working. Presenting a refresh token that was already used revokes every token from the same login (the "family") and returns 401. `POST /auth/logout` revokes the family of the given refresh token, and returns 204 even when the token is invalid, so it reveals nothing.

Field rules: `name` required, max 100 · `email` required, valid email, normalized to lower case · `password` required, min 8 (register only) · `role` one of `admin pengelola peminjam`.

### 8.2 Rooms

| Method & Path | Auth | Request body | Success `data` | Errors |
|---|---|---|---|---|
| `GET /rooms` | any | — | Room[] (active only, ordered by `code`) | 401 |
| `GET /rooms/:id` | any | — | Room (inactive rooms included) | 400, 401, 404 |
| `POST /rooms` | admin | RoomRequest | Room + header `Location: /api/v1/rooms/{id}` | 400, 401, 403, 409, 422 |
| `PUT /rooms/:id` | admin | RoomRequest | Room | 400, 401, 403, 404, 409, 422 |
| `DELETE /rooms/:id` | admin | — | *(204, no body)* | 400, 401, 403, 404 |
| `GET /rooms/:id/availability?date=YYYY-MM-DD` | any | — | Availability (only `pending` + `approved` bookings, ordered by `start_time`) | 400, 401, 404 |

RoomRequest:
```json
{ "code": "lab-01", "name": "Lab Komputer 1", "building": "Gedung C", "capacity": 40 }
```
`code` required, max 20, stored upper-case · `name` required, max 100 · `building` required, max 100 · `capacity` required, 1–1000.

### 8.3 Bookings

| Method & Path | Auth | Request body | Success `data` | Errors |
|---|---|---|---|---|
| `POST /bookings` | peminjam | CreateBookingRequest | Booking (`status: "pending"`) | 400, 401, 403, 404, 409, 422 |
| `PATCH /bookings/:id/approve` | pengelola | StatusChangeRequest (optional) | Booking | 400, 401, 403, 404, 409, 422 |
| `PATCH /bookings/:id/reject` | pengelola | StatusChangeRequest (optional) | Booking | 400, 401, 403, 404, 409, 422 |
| `PATCH /bookings/:id/cancel` | peminjam (owner) | StatusChangeRequest (optional) | Booking | 400, 401, 403, 404, 409, 422 |
| `GET /bookings/:id/logs` | owner or pengelola | — | BookingLog[] (oldest first) | 400, 401, 403, 404 |
| `GET /bookings/mine?status=` | peminjam | — | BookingDetail[] (newest `start_time` first) | 400, 401, 403 |
| `GET /bookings?status=&room_id=` | pengelola | — | BookingDetail[] (newest `start_time` first) | 400, 401, 403 |

CreateBookingRequest:
```json
{ "room_id": 1, "purpose": "Rapat BEM", "start_time": "2026-12-01T10:00:00+07:00", "end_time": "2026-12-01T12:00:00+07:00" }
```
StatusChangeRequest (the body may be empty or `{}`):
```json
{ "note": "Disetujui untuk kegiatan resmi" }
```
`note` max 255.

Status transitions (anything else → 409):

| From → To | Who |
|---|---|
| `pending → approved` | pengelola |
| `pending → rejected` | pengelola |
| `pending → cancelled` | peminjam who owns it |
| `approved → cancelled` | peminjam who owns it |

Every successful create or transition writes exactly one `booking_logs` row **in the same transaction**. The create row has `from_status: null`.

### 8.4 Notifications

| Method & Path | Auth | Request body | Success `data` | Errors |
|---|---|---|---|---|
| `GET /notifications?unread=true\|false` | peminjam | — | Notification[] (newest first; `unread=true` returns only unread) | 400, 401, 403 |
| `PATCH /notifications/:id/read` | peminjam | — | Notification | 400, 401, 403, 404 |
| `PATCH /notifications/read-all` | peminjam | — | `{ "updated": n }` | 401, 403 |

Notifications are created by the worker process, never by a request. Every booking creation or status change produces one notification for the booking owner, a moment after the booking response is sent. A user only ever sees their own notifications: another user's notification returns 404 on `PATCH /notifications/:id/read`. Register `/notifications/read-all` before `/notifications/:id/read`.

---

## 9. Implementation Rules

### 9.1 Service method template

Every `func(c *fiber.Ctx) error` follows this order. Don't skip or reorder steps.

```go
func (s *roomService) Update(c *fiber.Ctx) error {
    ctx, cancel := helper.RequestContext(c)            // 1. context with timeout
    defer cancel()

    id, ok := helper.ParamID(c, "id")                  // 2. path / query params → 400
    if !ok { return helper.BadRequest("id tidak valid") }

    var req model.RoomRequest                          // 3. body → 400
    if err := c.BodyParser(&req); err != nil { return helper.BadRequest("body request tidak valid") }

    if errs := helper.ValidateStruct(req); errs != nil { // 4. struct validation → 422
        return helper.Validation(errs)
    }

    room := model.Room{ID: id, Code: NormalizeRoomCode(req.Code), /* … */ }  // 5. rules

    updated, err := s.rooms.Update(ctx, room)          // 6. repository
    switch {                                           // 7. map errors (§9.3)
    case errors.Is(err, repository.ErrNotFound):  return helper.NotFound("ruang tidak ditemukan")
    case errors.Is(err, repository.ErrDuplicate): return helper.Conflict("kode ruang sudah digunakan")
    case err != nil:                              return helper.Internal(err)
    }
    return helper.Success(c, fiber.StatusOK, "ruang berhasil diperbarui", updated)  // 8. respond
}
```

Who is calling comes from `helper.CurrentUser(c)`, **never** from the body or the URL.

### 9.2 Repository rules

- Every query uses `$1, $2, …` placeholders. Building SQL with `fmt.Sprintf` or `+` on **values** is forbidden. A dynamic `WHERE` appends fixed fragments (`" AND status = $" + strconv.Itoa(n)`) and the value goes into `args`.
- MongoDB queries build filters from `bson` values and never by concatenating strings. A duplicate-key error maps to `ErrDuplicate`, and list methods return `[]T{}` just like SQL repositories.
- Each file declares its column list once as a constant (`const roomColumns = "id, code, name, building, capacity, is_active, created_at"`) and reuses it in `SELECT` and `RETURNING`.
- Every error passes through `TranslateError(err)` before it's returned.
- List methods return `[]T{}` when empty, never `nil`.
- A repository never starts a transaction. The service passes one in via `TxManager.Do` and `WithTx`.
- No business decisions in SQL beyond the constraints in `001_init.sql`.

### 9.3 Error mapping (repository → HTTP)

| Repository error | HTTP | Note |
|---|---|---|
| `ErrNotFound` | 404 | Use the resource-specific message |
| `ErrDuplicate` | 409 | |
| `ErrOverlap` | 409 | |
| `ErrConflict` | 409 | Guarded `UPDATE … WHERE status=$from` matched 0 rows |
| `ErrNotImplemented` | 501 | Skeleton only |
| anything else | 500 | `helper.Internal(err)`. The cause is logged, never sent to the client |

### 9.4 Authorization

- Route-level role checks go in `route.go` with `middleware.RequireRole`.
- Ownership checks ("is this my booking?") go in the **service**, using `helper.CurrentUser(c).ID`.
- A user who isn't allowed to see a resource gets 403 when the resource exists, and 404 when it doesn't.

### 9.5 Security

- Passwords are hashed with bcrypt (`helper.HashPassword`) and never logged or returned.
- Login returns the same 401 message for an unknown email and a wrong password.
- No secrets in code. Everything comes from env.
- The access JWT carries `sub` (user id), `email`, `role`, and `exp` only. Its lifetime is `JWT_TTL_MINUTES`.
- The refresh JWT carries `sub`, `jti` (unique token id), `fid` (family id), `typ: "refresh"`, and `exp` only. It is signed with `JWT_REFRESH_SECRET`, and every issued `jti` is recorded in PostgreSQL so a token can be used once and revoked.
- A refresh token is never accepted as a Bearer access token, and an access token is never accepted as a refresh token. Refresh tokens are never logged.

### 9.6 Background jobs

- Enqueue a job **after** `TxManager.Do` returns nil, never inside it, so a rolled-back change never produces a notification.
- A failed enqueue is logged and does not fail the request. The database change is the source of truth (known gap: there is no outbox).
- Jobs must be idempotent. The task ID is `booking-{id}-{status}`, MongoDB has a unique index on `(booking_id, status)`, and a duplicate-key error counts as success.
- The worker returns the error when saving fails, so asynq retries the job.
- The worker is a separate process. It never calls the API, and the API never calls the worker.

---

## 10. Testing Conventions

### 10.1 Unit tests (rules)

- Every `*_rules.go` has a `*_rules_test.go` in the same package.
- Tests are **table-driven** with `t.Run(tt.name, …)`. Case names are short Indonesian or English sentences: `"tepat bersambung tidak bentrok"`.
- Rules that depend on time receive `now` as a parameter. Tests never call `time.Now()`.
- Run with `go test ./...`. It must pass before you push.

### 10.2 HTTP test files (`http/*.http`, VS Code REST Client)

```http
@baseUrl = http://localhost:8080/api/v1
@adminToken = {{loginAdmin.response.body.data.access_token}}

### RM-03 create room as admin → 201, code stored as LAB-01
POST {{baseUrl}}/rooms
Authorization: Bearer {{adminToken}}
Content-Type: application/json

{ "code": "lab-01", "name": "Lab Komputer 1", "building": "Gedung C", "capacity": 40 }
```

- Every request title follows `### <ID> <what> → <expected status>`. The ID prefixes are `AU`, `RM`, `BK`, `HI`, `NT`.
- Tokens are captured with named requests (`# @name loginAdmin`), never pasted by hand.
- Each file runs top to bottom on a fresh database (`docker compose down -v && docker compose up -d`) and creates its own test data.

### 10.3 Integration and concurrency tests

- Files are named `*_concurrency_test.go` and start with `//go:build integration`.
- Run them with `go test -tags integration ./...` while the `docker compose` services are up, on a fresh database. They are not part of the default `go test ./...`.
- Each test asserts an exact outcome, for example 10 goroutines booking the same slot give exactly one 201 and nine 409.

---

## 11. Code Style

- Run `gofmt -w .` (or let the editor format on save) and `go vet ./...` before every push. A PR that isn't gofmt'd isn't reviewed.
- Keep functions under ~50 lines. If a service method grows past that, move the pure parts into `*_rules.go`.
- No commented-out code, no `fmt.Println` debugging, no `panic` outside `main.go` startup.
- Don't ignore errors (`_ = err`), except for `defer rows.Close()`-style cleanup.
- Comments explain **why**, not what.

---