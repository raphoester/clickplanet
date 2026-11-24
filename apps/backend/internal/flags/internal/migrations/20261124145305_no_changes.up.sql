-- A player changes its submission while it waits, never once the operator decided it: the founder's change of a flag
-- is gone. Its rows go. An approved one is on its flag already; a waiting or rejected one's picture stays private in
-- the bucket, under flags/submissions/<id>.png, and is never made public.
DELETE FROM submissions WHERE kind = 'change';

ALTER TABLE submissions DROP COLUMN kind;

DROP TYPE submission_kind;

-- Only a founder's change told a capital moved. The type keeps the value: postgres drops none.
DELETE FROM outbox WHERE event = 'flags.v1.FlagMoved';
