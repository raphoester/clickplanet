-- A shield taken from a hand and not yet saved on the map. The flush that saves its tile deletes the row, and a refusal
-- deletes it as it gives the shield back. A row left at boot is a shield the map never saved: the boot gives it back.
CREATE TABLE placements (
    id       uuid        PRIMARY KEY,
    account  uuid        NOT NULL,
    taken_at timestamptz NOT NULL
);
