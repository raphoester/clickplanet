-- The planet module's tile map. The map lives in memory; this is what a boot loads it from.
-- An unowned tile has no row, so the table holds only what somebody took.
CREATE TABLE tiles (
    id      integer PRIMARY KEY CHECK (id >= 0),
    country text    NOT NULL CHECK (country <> '')
);
