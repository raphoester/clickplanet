-- The schedule entry whose window a round of its own is, in the gamemaster's words. A day has none, so it is a table
-- of its own. A round of its own opened before the id was told has no row.
CREATE TABLE own_rounds (
    season   integer     NOT NULL,
    ends_at  timestamptz NOT NULL,
    own      boolean     NOT NULL CHECK (own),
    entry_id uuid        NOT NULL,
    PRIMARY KEY (season, ends_at, own),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);

-- A choice kept with a window and no entry is forgotten: the gamemaster commands its turn again at boot, with the id.
DELETE FROM rule_set_choice WHERE ends_at IS NOT NULL;
ALTER TABLE rule_set_choice
    ADD COLUMN entry_id uuid,
    ADD CONSTRAINT rule_set_choice_entry CHECK ((entry_id IS NULL) = (ends_at IS NULL));
