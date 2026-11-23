-- A flag opened since the game shipped wears a picture drawn again, named after its content, and ends. The table held
-- no row when this ran.
ALTER TABLE flags
    ADD COLUMN picture_hash    text        NOT NULL,
    ADD COLUMN picture_width   integer     NOT NULL,
    ADD COLUMN picture_height  integer     NOT NULL,
    ADD COLUMN picture_stretch boolean     NOT NULL,
    ADD COLUMN picture_focus_x real        NOT NULL,
    ADD COLUMN picture_focus_y real        NOT NULL,
    ADD COLUMN ends_at         timestamptz NOT NULL;

CREATE INDEX flags_open_ends_at ON flags (ends_at) WHERE state = 'open';
