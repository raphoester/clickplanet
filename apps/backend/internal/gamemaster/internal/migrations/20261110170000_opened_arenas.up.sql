-- Each arena planet told opened: its blank map plays, so the arena battle's round may open on it, from nothing.
CREATE TABLE opened_arenas (
    round_id text PRIMARY KEY CHECK (round_id <> '')
);
