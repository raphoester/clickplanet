-- What the operator writes to a player first, and what follows: a letter, its mails and the player's replies.
CREATE TABLE letters (
    id         uuid        PRIMARY KEY,
    -- Null once the letter is anonymized: its account was deleted, or it is past the retention.
    account_id uuid,
    -- Where its mails go; empty once anonymized.
    address    text        NOT NULL,
    subject    text        NOT NULL,
    -- Its first mail's.
    written_at timestamptz NOT NULL
);

CREATE INDEX letters_account ON letters (account_id);

CREATE INDEX letters_written_named ON letters (written_at) WHERE account_id IS NOT NULL;

CREATE TABLE letter_mails (
    letter_id  uuid        NOT NULL REFERENCES letters (id) ON DELETE CASCADE,
    message_id text        NOT NULL,
    text       text        NOT NULL,
    sent_at    timestamptz NOT NULL,
    PRIMARY KEY (letter_id, message_id)
);

CREATE TABLE letter_replies (
    letter_id   uuid        NOT NULL REFERENCES letters (id) ON DELETE CASCADE,
    message_id  text        NOT NULL,
    sender      text        NOT NULL,
    text        text        NOT NULL,
    received_at timestamptz NOT NULL,
    PRIMARY KEY (letter_id, message_id)
);

CREATE TABLE letter_closings (
    letter_id uuid        PRIMARY KEY REFERENCES letters (id) ON DELETE CASCADE,
    closed_at timestamptz NOT NULL
);
