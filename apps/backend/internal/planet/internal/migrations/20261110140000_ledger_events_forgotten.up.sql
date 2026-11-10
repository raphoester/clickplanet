-- 2026-11-10: a revert forgot its caller's events in the memory ledger alone, and only until they left the head, so
-- GetTileHolder named the reverted take on a tile the revert gave back. The events a mark covers say so now, for good:
-- the flush sets it with each new mark, and this fills it once from the marks still live.
ALTER TABLE ledger_events ADD COLUMN IF NOT EXISTS forgotten boolean NOT NULL DEFAULT false;

UPDATE ledger_events SET forgotten = true
FROM ledger_forgotten_accounts mark
WHERE ledger_events.account = mark.account
    AND ledger_events.position < mark.before_position;

UPDATE ledger_events SET forgotten = true
FROM ledger_forgotten mark
WHERE ledger_events.scope = mark.scope
    AND ledger_events.position >= (SELECT COALESCE(max(head), 0) FROM ledger_head)
    AND ledger_events.position < mark.before_position;
