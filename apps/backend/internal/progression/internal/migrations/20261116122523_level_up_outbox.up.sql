-- A level reached by playing waits here with the raise until Redis takes it: a level-up told is a level-up paid.
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'bonuses.v1.PayLevelUp';
