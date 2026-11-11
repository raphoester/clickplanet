-- The outbox told the arenas' open and close alone: the gamemaster calls planet for both now.
DELETE FROM outbox;
DROP TABLE outbox;
DROP TYPE outbox_event;
