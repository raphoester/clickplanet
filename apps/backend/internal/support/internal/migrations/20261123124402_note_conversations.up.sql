CREATE TABLE note_answers (
    note_id    uuid        NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    message_id text        NOT NULL,
    text       text        NOT NULL,
    sent_at    timestamptz NOT NULL,
    PRIMARY KEY (note_id, message_id)
);

CREATE TABLE note_replies (
    note_id     uuid        NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    message_id  text        NOT NULL,
    sender      text        NOT NULL,
    text        text        NOT NULL,
    received_at timestamptz NOT NULL,
    PRIMARY KEY (note_id, message_id)
);

CREATE TABLE note_closings (
    note_id   uuid        PRIMARY KEY REFERENCES notes (id) ON DELETE CASCADE,
    closed_at timestamptz NOT NULL
);
