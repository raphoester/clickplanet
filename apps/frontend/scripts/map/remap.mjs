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

const PAYLOAD_TILE_LISTS = ["cleared", "taken", "struck"]

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

export function remapSQL({runs, kept, removed, added, before, after, from, to}, {down = false} = {}) {
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

CREATE TEMP TABLE tiles_remapped ON COMMIT DROP AS
SELECT r.to_id + (t.id - r.from_id) AS id, t.country
FROM tiles t
JOIN tile_remap r ON t.id >= r.from_id AND t.id < r.from_id + r.span;

DELETE FROM tiles;
INSERT INTO tiles (id, country) SELECT id, country FROM tiles_remapped;

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

${PAYLOAD_TILE_LISTS.map(payloadTilesSQL).join("\n")}`
}
