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
