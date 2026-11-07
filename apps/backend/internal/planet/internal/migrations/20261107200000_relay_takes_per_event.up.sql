-- 2026-11-07: the takes and the shields struck go to new streams, one fact per ledger event (planet.v1.TilesTaken,
-- planet.v1.ShieldsStruck) instead of one per tile. What the old streams still held unread when the old server stopped
-- is told again on the new ones: the relay goes back to the first event of the last hour, and each receiver skips what
-- its marks counted already.
UPDATE ledger_relayed
SET position = LEAST(position, COALESCE(
    (SELECT min(position) FROM ledger_events WHERE taken_at >= now() - interval '1 hour'),
    position
));
