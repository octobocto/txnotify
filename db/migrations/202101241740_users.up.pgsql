CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid()
);