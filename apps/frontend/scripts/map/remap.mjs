/**
 * @param {string[]} before positions, as `keyOf` writes them, in the old blob's tile order
 * @param {string[]} after the same for the new blob
 * @returns {{
 *   runs: {from: number, to: number, span: number}[],
 *   kept: number, removed: number[], added: number,
 * }} runs in **wire ids**, 1-based; `removed` holds the wire ids the new blob has no tile for
 */
export function remap(before, after) {
    const oldIndexOf = new Map()
    for (let i = 0; i < before.length; i++) oldIndexOf.set(before[i], i)

    const runs = []
    const survived = new Set()
    let last = -1
    for (let i = 0; i < after.length; i++) {
        const old = oldIndexOf.get(after[i])
        if (old === undefined) continue
        if (old <= last) {
            throw new Error(
                `the mapping is not monotonic: new tile ${i + 1} came from old ${old + 1}, after ${last + 1}. `
                + "Both blobs must be the same lattice in generation order.",
            )
        }
        last = old
        survived.add(old)

        const open = runs[runs.length - 1]
        if (open && open.from + open.span === old + 1 && open.to + open.span === i + 1) {
            open.span++
            continue
        }
        runs.push({from: old + 1, to: i + 1, span: 1})
    }

    const removed = []
    for (let i = 0; i < before.length; i++) if (!survived.has(i)) removed.push(i + 1)

    return {runs, kept: survived.size, removed, added: after.length - survived.size}
}

/**
 * Which landmass of the new borders blob each landmass of the old one became: the one its surviving tiles are in,
 * on the same country's ground. A landmass whose tiles went to two (or that shares its new one with another) has none:
 * whatever was kept about it is about ground that is no longer one piece.
 * @param {{codes: string[], landmass: Uint16Array}} before
 * @param {{codes: string[], landmass: Uint16Array}} after
 * @param {{from: number, to: number, span: number}[]} runs the tiles' runs, in wire ids, from `remap`
 * @returns {{from: number, to: number}[]} by old id
 */
export function remapLandmasses(before, after, runs) {
    const into = new Map()
    for (const {from, to, span} of runs) {
        for (let k = 0; k < span; k++) {
            const old = before.landmass[from - 1 + k]
            const now = after.landmass[to - 1 + k]
            if (old === 0 || before.codes[old] !== after.codes[now]) continue
            if (!into.has(old)) into.set(old, new Set())
            into.get(old).add(now)
        }
    }

    const claims = new Map()
    for (const targets of into.values()) for (const now of targets) claims.set(now, (claims.get(now) ?? 0) + 1)

    const pairs = []
    for (const [old, targets] of into) {
        if (targets.size !== 1) continue
        const [now] = targets
        if (claims.get(now) === 1) pairs.push({from: old, to: now})
    }
    return pairs.sort((a, b) => a.from - b.from)
}

const PAYLOAD_TILE_LISTS = ["cleared", "taken", "struck", "raised"]

function payloadTilesSQL(key) {
    const tile = "(o.value->>'tile')::integer"
    return `UPDATE ledger_events e
SET payload = jsonb_set(e.payload, '{${key}}', COALESCE((
    SELECT jsonb_agg(jsonb_set(o.value, '{tile}', to_jsonb(r.to_id + (${tile} - r.from_id))) ORDER BY o.ordinality)
    FROM jsonb_array_elements(e.payload->'${key}') WITH ORDINALITY o
    JOIN tile_remap r ON ${tile} >= r.from_id AND ${tile} < r.from_id + r.span
), '[]'::jsonb))
WHERE e.payload ? '${key}';
`
}

export function remapSQL({runs, kept, removed, added, before, after, from, to, landmasses = [], borders}, {down = false} = {}) {
    const walked = down
        ? runs.map(({from: f, to: t, span}) => ({from: t, to: f, span}))
        : runs
    const values = walked.map(({from: f, to: t, span}) => `    (${f}, ${t}, ${span})`).join(",\n")
    const lost = down ? added : removed.length
    const gained = down ? removed.length : added

    return `-- Follows every owned tile from ${down ? to : from} to ${down ? from : to}: ${kept} of ${down ? after : before} kept, ${gained} unowned, ${lost} gone.

CREATE TEMP TABLE tile_remap (from_id integer NOT NULL, to_id integer NOT NULL, span integer NOT NULL)
ON COMMIT DROP;

INSERT INTO tile_remap (from_id, to_id, span) VALUES
${values};

CREATE INDEX ON tile_remap (from_id);

-- Every column of a tile goes with it, whatever columns the table has by the time this runs.
CREATE TEMP TABLE tiles_remapped ON COMMIT DROP AS
SELECT t.*, r.to_id + (t.id - r.from_id) AS remapped_id
FROM tiles t
JOIN tile_remap r ON t.id >= r.from_id AND t.id < r.from_id + r.span;

UPDATE tiles_remapped SET id = remapped_id;
ALTER TABLE tiles_remapped DROP COLUMN remapped_id;

DELETE FROM tiles;
INSERT INTO tiles SELECT * FROM tiles_remapped;

DELETE FROM ledger_events e
WHERE e.payload IS NULL AND NOT EXISTS (
    SELECT 1 FROM tile_remap r WHERE e.tile >= r.from_id AND e.tile < r.from_id + r.span
);

UPDATE ledger_events e
SET tile = 0, previous = ''
WHERE e.payload IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM tile_remap r WHERE e.tile >= r.from_id AND e.tile < r.from_id + r.span
);

UPDATE ledger_events e
SET tile = r.to_id + (e.tile - r.from_id)
FROM tile_remap r
WHERE e.tile >= r.from_id AND e.tile < r.from_id + r.span;

${PAYLOAD_TILE_LISTS.map(payloadTilesSQL).join("\n")}${borders ? landmassesSQL(landmasses, borders, down) : ""}`
}

// A landmass id is a row of the borders blob's table, so it moves with the blob too: the fortify a territory card
// reads last, and the lock the boot loads for the blob it runs on.
function landmassesSQL(pairs, borders, down) {
    const walked = down ? pairs.map(({from: f, to: t}) => ({from: t, to: f})) : pairs
    const [was, now] = down ? [borders.to, borders.from] : [borders.from, borders.to]
    const values = walked.map(({from: f, to: t}) => `    (${f}, ${t})`).join(",\n")

    return `
CREATE TEMP TABLE landmass_remap (from_id integer NOT NULL, to_id integer NOT NULL)
ON COMMIT DROP;
${values ? `
INSERT INTO landmass_remap (from_id, to_id) VALUES
${values};
` : ""}
UPDATE ledger_events e
SET payload = jsonb_set(e.payload, '{landmass}', to_jsonb(COALESCE((
    SELECT r.to_id FROM landmass_remap r WHERE r.from_id = (e.payload->>'landmass')::integer
), 0)))
WHERE e.payload ? 'landmass';

INSERT INTO landmasses (asset, id, fortified_by)
SELECT '${now}', r.to_id, l.fortified_by
FROM landmasses l
JOIN landmass_remap r ON r.from_id = l.id
WHERE l.asset = '${was}'
ON CONFLICT (asset, id) DO UPDATE SET fortified_by = EXCLUDED.fortified_by;
`
}
