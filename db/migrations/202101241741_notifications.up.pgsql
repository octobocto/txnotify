CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE if not exists notifications
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users (id),
    identifier    TEXT    NOT NULL,
    confirmations INTEGER NOT NULL,
    email         citext,
    description   TEXT
);
