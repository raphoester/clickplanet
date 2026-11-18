import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import {blobNamed, boardDir, readBoard, readBorders, staticDir} from "./board.mjs"
import {countryCodeOf} from "./ground.mjs"
import {framesOf} from "./landmasses.mjs"
import {lonLatOf} from "./lattice.mjs"
import {adminName} from "./names/labels.mjs"
import {clampLon, polygonIndex} from "./names/polygons.mjs"
import {naturalEarth} from "./naturalEarth.mjs"
import {positionOf} from "./seats.mjs"
import {tileLattice} from "./tileLattice.mjs"

export const UNITS_LAYER = "ne_10m_admin_1_states_provinces"

// The countries whose units name a nation of their own in `geonunit`: a province never crosses one.
const NATIONS = new Set(["GBR"])

/**
 * What the cut reads of each Natural Earth admin-1 unit.
 * @param {{features: {properties: Record<string, unknown>}[]}} layer
 * @returns {{code: string, country: string | null, name: string, region: string, nation: string}[]}
 */
export function unitsOf(layer) {
    const cleaned = (name) => String(name).replace(/\s*\([^)]*\)\s*/g, " ").replace(/\s+/g, " ").trim()
    const english = layer.features.map((feature) => cleaned(adminName(feature.properties)))
    const shared = new Map()
    layer.features.forEach((feature, f) => {
        const key = `${feature.properties.adm0_a3}|${english[f]}`
        shared.set(key, (shared.get(key) ?? 0) + 1)
    })
    return layer.features.map((feature, f) => {
        const p = feature.properties
        const region = String(p.region ?? "").trim()
        // Natural Earth gives some units the English name of another (Madagascar's regions, its old provinces').
        const alone = shared.get(`${p.adm0_a3}|${english[f]}`) === 1
        return {
            code: String(p.adm1_code),
            country: countryCodeOf(feature),
            name: alone || !p.name ? english[f] : cleaned(adminName({...p, name_en: null})),
            region: region && region !== "null" ? `${p.adm0_a3}|${region}` : "",
            nation: NATIONS.has(String(p.adm0_a3)) ? String(p.geonunit) : "",
        }
    })
}

/**
 * Each tile's unit: the one under its centre in the tile's own country, else none (-1).
 * @param {{count: number, positions: Float32Array, codes: string[], landmass: Uint16Array}} board
 * @param {{features: object[]}} layer
 * @param {ReturnType<typeof unitsOf>} units
 */
export function unitsUnder(board, layer, units) {
    const index = polygonIndex(layer.features)
    const unit = new Int32Array(board.count).fill(-1)
    let foreign = 0
    let none = 0
    for (let t = 0; t < board.count; t++) {
        const [lon, lat] = lonLatOf(board.positions, t)
        const hits = index.contains(clampLon(lon), lat)
        const code = board.codes[board.landmass[t]]
        const own = hits.find((f) => units[f].country === code)
        if (own !== undefined) unit[t] = own
        else if (hits.length > 0) foreign++
        else none++
    }
    return {unit, foreign, none}
}

/**
 * A board's provinces: each tile takes a unit of its own country, the touching tiles of one unit inside one landmass
 * are a piece, and a piece under `floor` tiles joins a touching piece of its landmass, the smallest first.
 * @param {{
 *   count: number,
 *   positions: Float32Array,
 *   points: Uint32Array,
 *   codes: string[],
 *   landmass: Uint16Array,
 *   at: Uint32Array,
 *   to: Uint32Array,
 *   tileOf: Int32Array,
 * }} board
 * @param {Int32Array} under each tile's unit, from `unitsUnder`
 * @param {ReturnType<typeof unitsOf>} units
 * @param {number} floor N: the fewest tiles a province has, unless it is a whole landmass or a whole nation
 */
export function provincesOf(board, under, units, floor) {
    const {count, landmass} = board
    const neighboursOf = (t, visit) => {
        const v = board.points[t]
        for (let e = board.at[v]; e < board.at[v + 1]; e++) {
            const n = board.tileOf[board.to[e]]
            if (n >= 0 && landmass[n] === landmass[t]) visit(n)
        }
    }

    const unit = Int32Array.from(under)
    const queue = new Int32Array(count)
    let head = 0
    let tail = 0
    for (let t = 0; t < count; t++) if (unit[t] >= 0) queue[tail++] = t
    const filled = count - tail
    while (head < tail) {
        const t = queue[head++]
        neighboursOf(t, (n) => {
            if (unit[n] >= 0) return
            unit[n] = unit[t]
            queue[tail++] = n
        })
    }

    const piece = new Int32Array(count).fill(-1)
    const pieces = []
    for (let t = 0; t < count; t++) {
        if (piece[t] >= 0) continue
        const id = pieces.length
        const own = {id, landmass: landmass[t], unit: unit[t], tiles: 0, first: t}
        pieces.push(own)
        piece[t] = id
        head = 0
        tail = 0
        queue[tail++] = t
        while (head < tail) {
            const at = queue[head++]
            own.tiles++
            neighboursOf(at, (n) => {
                if (piece[n] >= 0 || unit[n] !== own.unit) return
                piece[n] = id
                queue[tail++] = n
            })
        }
    }

    const groups = pieces.map((p) => ({
        id: p.id,
        landmass: p.landmass,
        first: p.first,
        tiles: p.tiles,
        region: p.unit >= 0 ? units[p.unit].region : "",
        nation: p.unit >= 0 ? units[p.unit].nation : "",
        units: new Map(p.unit >= 0 ? [[p.unit, p.tiles]] : []),
        touching: new Map(),
        alive: true,
    }))
    for (let t = 0; t < count; t++) {
        neighboursOf(t, (n) => {
            if (n <= t || piece[n] === piece[t]) return
            const [a, b] = [groups[piece[t]], groups[piece[n]]]
            a.touching.set(b.id, (a.touching.get(b.id) ?? 0) + 1)
            b.touching.set(a.id, (b.touching.get(a.id) ?? 0) + 1)
        })
    }

    const heap = new MinHeap()
    for (const group of groups) if (group.tiles < floor) heap.push(group.tiles, group.id)
    let merges = 0
    while (heap.size > 0) {
        const [tiles, id] = heap.pop()
        const small = groups[id]
        if (!small.alive || small.tiles !== tiles) continue
        let best
        let bestKey
        for (const [otherId, shared] of small.touching) {
            const other = groups[otherId]
            if (other.nation !== small.nation) continue
            const key = [small.region !== "" && other.region === small.region ? 1 : 0, other.tiles < floor ? 1 : 0, shared, -other.id]
            if (bestKey === undefined || compareKeys(key, bestKey) > 0) {
                best = other
                bestKey = key
            }
        }
        if (best === undefined) continue
        absorb(groups, best, small)
        merges++
        if (best.tiles < floor) heap.push(best.tiles, best.id)
    }

    const kept = groups.filter((group) => group.alive)
        .sort((a, b) => a.landmass - b.landmass || a.first - b.first)
    const provinceOfGroup = new Map(kept.map((group, k) => [group.id, k + 1]))
    if (kept.length > 65535) throw new Error(`${kept.length} provinces, more than a uint16 names`)

    const groupOfPiece = new Int32Array(pieces.length)
    for (const group of groups) if (group.alive) for (const p of group.pieces ?? [group.id]) groupOfPiece[p] = group.id
    const province = new Uint16Array(count)
    for (let t = 0; t < count; t++) province[t] = provinceOfGroup.get(groupOfPiece[piece[t]])

    const members = Array.from({length: kept.length + 1}, () => [])
    for (let t = 0; t < count; t++) members[province[t]].push(t)
    const landmasses = new Uint16Array(kept.length + 1)
    const totals = new Uint32Array(kept.length + 1)
    kept.forEach((group, k) => {
        landmasses[k + 1] = group.landmass
        totals[k + 1] = group.tiles
    })

    return {
        province,
        landmasses,
        totals,
        frames: framesOf(board.positions, members),
        units: [undefined, ...kept.map((group) => group.units)],
        nations: ["", ...kept.map((group) => group.nation)],
        stats: {filled, pieces: pieces.length, merges},
    }
}

function absorb(groups, into, small) {
    into.tiles += small.tiles
    for (const [unit, tiles] of small.units) into.units.set(unit, (into.units.get(unit) ?? 0) + tiles)
    into.pieces = [...(into.pieces ?? [into.id]), ...(small.pieces ?? [small.id])]
    for (const [otherId, shared] of small.touching) {
        if (otherId === into.id) continue
        const other = groups[otherId]
        other.touching.delete(small.id)
        other.touching.set(into.id, (other.touching.get(into.id) ?? 0) + shared)
        into.touching.set(otherId, (into.touching.get(otherId) ?? 0) + shared)
    }
    into.touching.delete(small.id)
    small.alive = false
    small.touching = new Map()
}

function compareKeys(a, b) {
    for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return a[i] - b[i]
    return 0
}

// By tiles, then by id: the smallest first, and among equals the first piece.
class MinHeap {
    #items = []

    get size() {
        return this.#items.length
    }

    push(tiles, id) {
        const items = this.#items
        items.push([tiles, id])
        let i = items.length - 1
        while (i > 0) {
            const parent = (i - 1) >> 1
            if (compareKeys(items[parent], items[i]) <= 0) break
            ;[items[parent], items[i]] = [items[i], items[parent]]
            i = parent
        }
    }

    pop() {
        const items = this.#items
        const top = items[0]
        const last = items.pop()
        if (items.length > 0) {
            items[0] = last
            let i = 0
            for (;;) {
                const [l, r] = [i * 2 + 1, i * 2 + 2]
                let least = i
                if (l < items.length && compareKeys(items[l], items[least]) < 0) least = l
                if (r < items.length && compareKeys(items[r], items[least]) < 0) least = r
                if (least === i) break
                ;[items[least], items[i]] = [items[i], items[least]]
                i = least
            }
        }
        return top
    }
}

/**
 * Each province's English name, by id: one unit's name, a whole region's name, or its largest unit's, unless an
 * override names its exact units (and, for two provinces of the same units, the one nearest its point `at`). A province
 * that is a whole landmass is named as the landmass, and numbered as the landmasses are ("Texas 2") when another
 * province of its country has that name; one that is its country's whole main landmass has no name of its own, and reads
 * as its country, as the landmass does.
 * @param {ReturnType<typeof provincesOf>} cut
 * @param {ReturnType<typeof unitsOf>} units
 * @param {Override[]} overrides
 * @param {{codes: string[], landmassNames: Map<number, {name: string, main: boolean}>}} landmasses each landmass's
 *   country and name
 * @returns {{names: Map<number, string>, alike: string[], overridden: number, regions: number, largest: number,
 *   wholes: number, countries: number, renamed: number}} `alike`, the provinces of one country that read the same, or have no name, which
 *   an override must name
 * @typedef {{name: string, units: Record<string, string>, at?: [number, number]}} Override
 */
export function provinceNamesOf(cut, units, overrides, {codes, landmassNames}) {
    const indexOf = new Map(units.map((unit, f) => [unit.code, f]))
    const keyOf = (k) => [...cut.units[k].keys()].map((f) => units[f].code).sort().join(" ")
    const provincesOf = new Map()
    for (let k = 1; k < cut.landmasses.length; k++) provincesOf.set(keyOf(k), [...(provincesOf.get(keyOf(k)) ?? []), k])

    const overrideOf = new Map()
    for (const override of overrides) {
        const codes = Object.keys(override.units).sort()
        for (const code of codes) {
            const f = indexOf.get(code)
            if (f === undefined) throw new Error(`override "${override.name}": no unit ${code}`)
            if (units[f].name !== override.units[code]) {
                throw new Error(`override "${override.name}": ${code} is "${units[f].name}", not "${override.units[code]}"`)
            }
        }
        const matched = provincesOf.get(codes.join(" ")) ?? []
        if (matched.length === 0) throw new Error(`override "${override.name}" matches no province`)
        if (matched.length > 1 && override.at === undefined) {
            throw new Error(`override "${override.name}" matches ${matched.length} provinces: give it a point, \`at\``)
        }
        const k = override.at === undefined ? matched[0] : nearestTo(cut.frames, matched, positionOf(override.at))
        if (overrideOf.has(k)) throw new Error(`overrides "${overrideOf.get(k).name}" and "${override.name}" name one province`)
        overrideOf.set(k, override)
    }

    const provincesOn = new Map()
    const regionProvincesOn = new Map()
    for (let k = 1; k < cut.landmasses.length; k++) {
        const on = cut.landmasses[k]
        provincesOn.set(on, (provincesOn.get(on) ?? 0) + 1)
        for (const f of cut.units[k].keys()) {
            const key = `${on}|${units[f].region}`
            if (!regionProvincesOn.has(key)) regionProvincesOn.set(key, new Set())
            regionProvincesOn.get(key).add(k)
        }
    }

    const names = new Map()
    const wholes = []
    let regions = 0
    let largest = 0
    for (let k = 1; k < cut.landmasses.length; k++) {
        const own = cut.units[k]
        if (overrideOf.has(k)) {
            names.set(k, overrideOf.get(k).name)
            continue
        }
        if (provincesOn.get(cut.landmasses[k]) === 1) {
            wholes.push(k)
            continue
        }
        if (own.size === 1) {
            names.set(k, units[[...own.keys()][0]].name)
            continue
        }
        const region = units[[...own.keys()][0]].region
        const holders = regionProvincesOn.get(`${cut.landmasses[k]}|${region}`)
        if (region !== "" && [...own.keys()].every((f) => units[f].region === region) && holders.size === 1) {
            names.set(k, region.slice(region.indexOf("|") + 1))
            regions++
            continue
        }
        const [biggest] = [...own].sort(([a, x], [b, y]) => y - x || a - b)[0]
        names.set(k, units[biggest].name)
        largest++
    }

    const alike = []
    const taken = new Map()
    const keyOfName = (k, name) => `${codes[cut.landmasses[k]]} ${name.toLowerCase()}`
    for (const [k, name] of names) {
        if (!name.trim()) alike.push(`${codes[cut.landmasses[k]]} province ${k} has no name`)
        else if (taken.has(keyOfName(k, name))) alike.push(`${keyOfName(k, name)} (${taken.get(keyOfName(k, name))}, ${k})`)
        else taken.set(keyOfName(k, name), k)
    }
    const renamed = []
    let countries = 0
    for (const k of wholes) {
        const {name, main} = landmassNames.get(cut.landmasses[k])
        if (main && taken.has(keyOfName(k, name))) alike.push(`${keyOfName(k, name)}, its country (${taken.get(keyOfName(k, name))}, ${k})`)
        else if (taken.has(keyOfName(k, name))) renamed.push(k)
        else taken.set(keyOfName(k, name), k)
        if (main) countries++
        else names.set(k, name)
    }
    for (const k of renamed) {
        const base = names.get(k)
        let n = 2
        while (taken.has(keyOfName(k, `${base} ${n}`))) n++
        names.set(k, `${base} ${n}`)
        taken.set(keyOfName(k, names.get(k)), k)
    }
    return {names, alike, overridden: overrideOf.size, regions, largest, wholes: wholes.length - countries, countries, renamed: renamed.length}
}

function nearestTo(frames, provinces, [x, y, z]) {
    let best = provinces[0]
    let bestDot = -2
    for (const k of provinces) {
        const dot = frames[k * 5] * x + frames[k * 5 + 1] * y + frames[k * 5 + 2] * z
        if (dot > bestDot) {
            best = k
            bestDot = dot
        }
    }
    return best
}

/**
 * A board's provinces at a floor, from its files and Natural Earth's units.
 * @param {string} name the board's
 * @param {number} floor
 */
export async function cutOf(name, floor) {
    const board = readBoard(boardDir(name))
    const borders = readBorders(board)
    const layer = await naturalEarth(UNITS_LAYER)
    const units = unitsOf(layer)
    const ground = {...board, ...tileLattice(board), codes: borders.codes, landmass: borders.landmass}
    const under = unitsUnder(ground, layer, units)
    return {board, borders, units, under, cut: provincesOf(ground, under.unit, units, floor)}
}

/**
 * Each landmass's name as the page reads it: its own, or its country's for the country's main one.
 * @param {string} board
 * @param {{name: string, codes: string[]}} borders
 * @returns {Map<number, {name: string, main: boolean}>}
 */
export function landmassNamesOf(board, borders) {
    const directory = path.join(staticDir, "boards", board)
    const file = blobNamed(/^landmassNames-[0-9a-f]{8}\.json$/, directory)
    const own = JSON.parse(fs.readFileSync(path.join(directory, file), "utf8"))
    if (own.borders !== borders.name) {
        throw new Error(`${file} names ${own.borders}'s landmasses, not ${borders.name}'s: run \`npm run landmassNames -- ${board}\``)
    }
    const countries = JSON.parse(fs.readFileSync(path.join(staticDir, "countries", "countries.json"), "utf8"))
    const names = new Map()
    for (let k = 1; k < borders.codes.length; k++) {
        const name = own.names[String(k)]
        names.set(k, name === undefined ? {name: countries[borders.codes[k]] ?? borders.codes[k], main: true} : {name, main: false})
    }
    return names
}

/**
 * The names given by hand to a board's provinces (scripts/map/provinceNames/<board>.json).
 * @returns {Override[]}
 */
export function overridesOf(board) {
    const file = fileURLToPath(new URL(`./provinceNames/${board}.json`, import.meta.url))
    return fs.existsSync(file) ? JSON.parse(fs.readFileSync(file, "utf8")) : []
}
