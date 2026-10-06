DROP TABLE garrisons;

DELETE FROM charges WHERE NOT refill AND NOT bomb AND enclosures = 0 AND spread_clicks = 0;
ALTER TABLE charges DROP COLUMN defenders;
