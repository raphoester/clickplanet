-- The library of the teasers' backgrounds: each file is in the bucket under promo/backgrounds/<background_id>, and
-- public on the media domain.
CREATE TABLE backgrounds (
    background_id uuid        PRIMARY KEY,
    name          text        NOT NULL CHECK (name <> ''),
    media_type    text        NOT NULL,
    size_bytes    bigint      NOT NULL CHECK (size_bytes > 0),
    kept_at       timestamptz NOT NULL
);

CREATE INDEX backgrounds_kept_at ON backgrounds (kept_at DESC);

-- Each rule set's cards, in order, under the set's name: planet and the antibot keep their rules under the same
-- names. A background a card shows cannot be removed.
CREATE TABLE cards (
    rule_set      text     NOT NULL CHECK (rule_set <> ''),
    position      smallint NOT NULL CHECK (position BETWEEN 0 AND 4),
    background_id uuid     NOT NULL REFERENCES backgrounds (background_id),
    headline      text     NOT NULL CHECK (headline <> ''),
    line          text     NOT NULL,
    PRIMARY KEY (rule_set, position)
);

CREATE INDEX cards_background_id ON cards (background_id);
