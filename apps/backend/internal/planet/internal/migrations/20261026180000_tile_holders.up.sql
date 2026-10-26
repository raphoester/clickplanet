ALTER TABLE tiles ADD COLUMN IF NOT EXISTS holder uuid;

-- The account of the last change the ledger kept of each tile, as GetTileHolder reads it, while that change gave the tile
-- the flag it wears: a tile an operator wrote since keeps no holder.
WITH last_take AS (
    SELECT DISTINCT ON (tile) tile, position, account, country
    FROM ledger_events
    WHERE kind = 'take'
    ORDER BY tile DESC, position DESC
), payload_changes AS (
    SELECT (taken ->> 'tile')::integer AS tile, position, account, country
    FROM ledger_events, jsonb_array_elements(payload -> 'taken') AS taken
    WHERE kind IN ('spread', 'enclose')
    UNION ALL
    SELECT (cleared ->> 'tile')::integer, position, NULL::uuid, ''
    FROM ledger_events, jsonb_array_elements(payload -> 'cleared') AS cleared
    WHERE kind = 'bomb'
), last_change AS (
    SELECT DISTINCT ON (tile) tile, account, country
    FROM (SELECT * FROM last_take UNION ALL SELECT * FROM payload_changes) AS changes
    ORDER BY tile, position DESC
)
UPDATE tiles SET holder = last_change.account
FROM last_change
WHERE tiles.id = last_change.tile AND tiles.country = last_change.country AND last_change.account IS NOT NULL;
