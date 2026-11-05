ALTER TABLE stats
    ADD COLUMN bombs_dropped bigint NOT NULL DEFAULT 0 CHECK (bombs_dropped >= 0),
    ADD COLUMN tiles_fortified bigint NOT NULL DEFAULT 0 CHECK (tiles_fortified >= 0),
    ADD COLUMN bonuses_claimed bigint NOT NULL DEFAULT 0 CHECK (bonuses_claimed >= 0),
    ADD COLUMN right_answers bigint NOT NULL DEFAULT 0 CHECK (right_answers >= 0);
