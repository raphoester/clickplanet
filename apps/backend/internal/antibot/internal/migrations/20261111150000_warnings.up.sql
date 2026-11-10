CREATE TABLE warnings (
    account   uuid        PRIMARY KEY,
    warned_at timestamptz NOT NULL
);
