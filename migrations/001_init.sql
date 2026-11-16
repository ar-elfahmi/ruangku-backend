-- Skema awal RuangKu. Dijalankan otomatis oleh postgres lewat
-- /docker-entrypoint-initdb.d saat volume database masih kosong.

CREATE TABLE users (
    id            SERIAL       PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'peminjam',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT users_role_check CHECK (role IN ('admin', 'pengelola', 'peminjam'))
);

-- Email unik tanpa membedakan huruf besar/kecil.
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

CREATE TABLE rooms (
    id         SERIAL       PRIMARY KEY,
    code       VARCHAR(20)  NOT NULL,
    name       VARCHAR(100) NOT NULL,
    building   VARCHAR(100) NOT NULL,
    capacity   INT          NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT rooms_code_key UNIQUE (code),
    CONSTRAINT rooms_capacity_check CHECK (capacity BETWEEN 1 AND 1000)
);

-- Dibutuhkan agar EXCLUDE bisa membandingkan room_id (integer) dengan operator "=".
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE bookings (
    id         SERIAL       PRIMARY KEY,
    user_id    INT          NOT NULL,
    room_id    INT          NOT NULL,
    purpose    VARCHAR(255) NOT NULL,
    start_time TIMESTAMPTZ  NOT NULL,
    end_time   TIMESTAMPTZ  NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT bookings_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT bookings_room_id_fkey FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE RESTRICT,
    CONSTRAINT bookings_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    CONSTRAINT bookings_time_check CHECK (start_time < end_time),
    -- Interval setengah terbuka [): 10-12 lalu 12-14 tidak bentrok. Hanya booking
    -- pending/approved yang memblokir ruang. Constraint ini menjamin dua request
    -- bersamaan untuk slot yang sama tidak bisa sama-sama berhasil.
    CONSTRAINT bookings_room_time_excl EXCLUDE USING gist (
        room_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    ) WHERE (status IN ('pending', 'approved'))
);

CREATE INDEX bookings_user_id_idx ON bookings (user_id);

CREATE TABLE booking_logs (
    id          SERIAL       PRIMARY KEY,
    booking_id  INT          NOT NULL,
    actor_id    INT          NOT NULL,
    from_status VARCHAR(20),
    to_status   VARCHAR(20)  NOT NULL,
    note        VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT booking_logs_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES bookings (id) ON DELETE CASCADE,
    CONSTRAINT booking_logs_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT booking_logs_from_status_check CHECK (from_status IN ('pending', 'approved', 'rejected', 'cancelled')),
    CONSTRAINT booking_logs_to_status_check CHECK (to_status IN ('pending', 'approved', 'rejected', 'cancelled'))
);

CREATE INDEX booking_logs_booking_id_idx ON booking_logs (booking_id);
