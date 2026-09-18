-- A charge is kept until it is used: nothing lapses, so a refill and a bomb are a flag rather than a time,
-- and enclosures a count, as spread clicks already were. A charge still in time when this runs is kept; a
-- row left holding nothing goes.
ALTER TABLE charges
    ADD COLUMN refill  boolean NOT NULL DEFAULT false,
    ADD COLUMN bomb    boolean NOT NULL DEFAULT false,
    ADD COLUMN enclosures integer NOT NULL DEFAULT 0 CHECK (enclosures >= 0);

UPDATE charges SET
    bomb          = coalesce(bomb_until > now(), false),
    enclosures    = CASE WHEN enclose_until > now() THEN 1 ELSE 0 END,
    spread_clicks = CASE WHEN spread_until > now() THEN spread_clicks ELSE 0 END;

ALTER TABLE charges
    DROP COLUMN bomb_until,
    DROP COLUMN enclose_until,
    DROP COLUMN spread_until;

DELETE FROM charges WHERE NOT refill AND NOT bomb AND enclosures = 0 AND spread_clicks = 0;
