ALTER TABLE charges
    ADD COLUMN bomb_until    timestamptz,
    ADD COLUMN enclose_until timestamptz,
    ADD COLUMN spread_until  timestamptz;

UPDATE charges SET
    bomb_until    = CASE WHEN bomb THEN now() + interval '24 hours' END,
    enclose_until = CASE WHEN enclosures > 0 THEN now() + interval '24 hours' END,
    spread_until  = CASE WHEN spread_clicks > 0 THEN now() + interval '24 hours' END;

ALTER TABLE charges
    DROP COLUMN refill,
    DROP COLUMN bomb,
    DROP COLUMN enclosures;
