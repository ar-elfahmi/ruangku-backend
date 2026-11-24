CREATE TABLE refresh_families (
    id             VARCHAR(32) PRIMARY KEY,
    user_id        INT         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at     TIMESTAMPTZ,
    revoked_reason VARCHAR(20),
    CONSTRAINT refresh_families_reason_check CHECK (revoked_reason IN ('logout', 'reuse'))
);

CREATE TABLE refresh_tokens (
    jti         VARCHAR(32) PRIMARY KEY,
    family_id   VARCHAR(32) NOT NULL REFERENCES refresh_families (id) ON DELETE CASCADE,
    user_id     INT         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    issued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    replaced_by VARCHAR(32)
);

CREATE INDEX refresh_tokens_family_id_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_user_id_idx   ON refresh_tokens (user_id);