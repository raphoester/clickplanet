DROP TABLE charge_fills;
DROP TYPE charge_fill_source;

ALTER TABLE charges DROP COLUMN last_fill;
DELETE FROM charges WHERE NOT refill AND NOT bomb AND enclosures = 0 AND spread_clicks = 0 AND shields = 0;
