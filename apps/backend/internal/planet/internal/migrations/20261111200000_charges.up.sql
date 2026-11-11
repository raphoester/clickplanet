-- The wallet's half of what each set held under "bonuses" moves to "charges" (planet.v1.ChargeRules): spread.clicks,
-- bomb.rings and bomb.off, enclose.maxTiles, held and unlimited, shield.held and perTile. The offers' half, their rules
-- now, is the bonuses module's: its first migration copies it from "bonuses", which stays here for that copy whichever
-- module migrates first, and which planet no longer reads (Rules.bonuses is reserved). A later release strips it.
UPDATE rule_sets
SET rules = rules || jsonb_build_object('charges', wallet)
FROM (
    SELECT name AS kept, jsonb_strip_nulls(jsonb_build_object(
        'spread', CASE WHEN b -> 'spread' ? 'clicks'
            THEN jsonb_build_object('clicks', b -> 'spread' -> 'clicks') END,
        'bomb', CASE WHEN b -> 'bomb' ?| ARRAY['rings', 'off']
            THEN jsonb_build_object('rings', b -> 'bomb' -> 'rings', 'off', b -> 'bomb' -> 'off') END,
        'enclose', CASE WHEN b -> 'enclose' ?| ARRAY['maxTiles', 'held', 'unlimited']
            THEN jsonb_build_object('maxTiles', b -> 'enclose' -> 'maxTiles', 'held', b -> 'enclose' -> 'held',
                'unlimited', b -> 'enclose' -> 'unlimited') END,
        'shield', CASE WHEN b -> 'shield' ?| ARRAY['held', 'perTile']
            THEN jsonb_build_object('held', b -> 'shield' -> 'held', 'perTile', b -> 'shield' -> 'perTile') END
    )) AS wallet
    FROM (SELECT name, COALESCE(rules -> 'bonuses', '{}'::jsonb) AS b FROM rule_sets) held
) halves
WHERE rule_sets.name = halves.kept AND halves.wallet <> '{}'::jsonb;
