CREATE TYPE note_kind AS ENUM ('suggestion');

CREATE TABLE notes (
    id           uuid        PRIMARY KEY,
    kind         note_kind   NOT NULL,
    account_id   uuid        NOT NULL,
    text         text        NOT NULL,
    submitted_at timestamptz NOT NULL
);

CREATE INDEX notes_submitted_at ON notes (submitted_at);

CREATE INDEX notes_account ON notes (account_id);
