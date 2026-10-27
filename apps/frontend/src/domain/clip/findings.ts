import {ranked, tally, TileChange} from "./changes.ts"
import {Point} from "./geometry.ts"
import {Flip, flippedTo} from "./look.ts"
import {solidityOf} from "./solidity.ts"
import {Place, RegionOf, Story, THE_WORLD} from "./story.ts"

// The map at one moment: each tile's holder, by tile id.
export type Owners = ReadonlyMap<number, string>

// Flags and a count each, the most first.
export type Counts = Record<string, number>

// The ground: which country a tile is on, the tiles of a country, and the continent of a country.
export type Ground = {
    groundOf: (tile: number) => string | undefined
    tilesOf: (country: string) => readonly number[]
    regionOf: RegionOf
}

export type FrontCountry = {
    code: string
    continent: string | undefined
    tiles: number
    changed: number
    holdersStart: Counts
    holdersEnd: Counts
    // The flags that took the most from another flag there.
    takers: Counts
}

export type FrontFlag = {
    code: string
    continent: string | undefined
    // Tiles taken from another flag, taken off empty ground, and lost to another flag.
    took: number
    filled: number
    lost: number
    tookFrom: Counts
    // Of what it took and holds at the end; 1 for a flag that took nothing.
    solidity: number
}

export type FrontFacts = {
    tilesChanged: number
    captured: number
    countries: FrontCountry[]
    flags: FrontFlag[]
}

export type TourFacts = {
    took: number
    byContinent: Counts
    tookFrom: Counts
    solidity: number
    // The tiles of the landmasses whose painted flag switched to the flag, by continent: all a tour shows from far.
    flippedByContinent: Counts
}

// The flag holding the most of a place as the map stands, and what changed hands to it or from it there in the window.
export type PlaceFacts = {
    kind: "world" | "continent" | "country"
    place: Place
    code: string | undefined
    continent: string | undefined
    tiles: number
    holder: string
    held: number
    share: number
    rivals: Counts
    changed: number
}

const MOST_COUNTRIES = 6

const MOST_FLAGS = 6

const MOST_HOLDERS = 3

// Solidity is measured for the flags that took the most: it reads every tile a flag holds.
const SOLID_FLAGS = 3

// Russia's ground is 71% of Natural Earth's Europe: a flag holding Siberia does not hold Europe. Russia is a place of
// its own, and Europe is held without it.
const OUTSIDE_EUROPE = "ru"

// About a France: 49 countries are this big, and a smaller one held by a foreign flag is a corner of a war.
const BIG_COUNTRY = 1000

const RIVALS = 2

export function top(counts: ReadonlyMap<string, number>, most: number): Counts {
    return Object.fromEntries(ranked(counts).slice(0, most))
}

function sumBy(entries: Iterable<readonly [string, number]>): Map<string, number> {
    const sums = new Map<string, number>()
    for (const [key, count] of entries) sums.set(key, (sums.get(key) ?? 0) + count)
    return sums
}

// How solid the land a flag took is, against all it holds at the end. taken: the tiles it took; held: every tile
// it holds.
export function solidityAfter(flag: string, changes: readonly TileChange[], after: Owners, pointOf: (tile: number) => Point): number {
    const held: Point[] = []
    for (const [tile, owner] of after) if (owner === flag) held.push(pointOf(tile))
    const taken = [...new Set(changes.flatMap(({tile, to}) => to === flag && after.get(tile) === flag ? [tile] : []))]
    return solidityOf(taken.map(pointOf), held)
}

// What a front is, measured and not judged: where its tiles changed hands and who held each country at its start and
// its end, who took and lost how much there and from whom. opening and after: the map when the front opens and when it
// ends.
export function frontFactsOf(
    changes: readonly TileChange[],
    opening: Owners,
    after: Owners,
    ground: Ground,
    pointOf: (tile: number) => Point,
): FrontFacts {
    const moves = changes.filter(({from, to}) => from !== to)
    const captured = moves.filter((change): change is TileChange & {from: string, to: string} =>
        change.from !== undefined && change.to !== undefined)

    const countries = ranked(tally(moves.flatMap(({tile}) => ground.groundOf(tile) ?? []))).slice(0, MOST_COUNTRIES)
        .map(([code, changed]): FrontCountry => {
            const tiles = ground.tilesOf(code)
            const holders = (owners: Owners) => top(tally(tiles.flatMap((tile) => owners.get(tile) ?? [])), MOST_HOLDERS)
            return {
                code, continent: ground.regionOf(code), tiles: tiles.length, changed,
                holdersStart: holders(opening), holdersEnd: holders(after),
                takers: top(tally(captured.flatMap(({tile, to}) => ground.groundOf(tile) === code ? [to] : [])), MOST_HOLDERS),
            }
        })

    const took = tally(captured.map(({to}) => to))
    const filled = tally(moves.flatMap(({from, to}) => from === undefined && to !== undefined ? [to] : []))
    const lost = tally(captured.map(({from}) => from))
    const involved = new Map<string, number>()
    for (const counts of [took, lost]) {
        for (const [flag, count] of counts) involved.set(flag, (involved.get(flag) ?? 0) + count)
    }
    const takers = ranked(took).slice(0, SOLID_FLAGS).map(([flag]) => flag)
    const flags = ranked(involved).slice(0, MOST_FLAGS).map(([code]): FrontFlag => ({
        code, continent: ground.regionOf(code), took: took.get(code) ?? 0, filled: filled.get(code) ?? 0, lost: lost.get(code) ?? 0,
        tookFrom: top(tally(captured.flatMap(({from, to}) => to === code ? [from] : [])), MOST_HOLDERS),
        solidity: takers.includes(code) ? solidityAfter(code, moves, after, pointOf) : 1,
    }))

    return {tilesChanged: moves.length, captured: captured.length, countries, flags}
}

// A flag's taking across the world in a window: changes is what it took and lost there, flips the landmasses whose
// biggest holder changed along them.
export function tourFactsOf(
    flag: string,
    changes: readonly TileChange[],
    after: Owners,
    continentOf: (tile: number) => string | undefined,
    pointOf: (tile: number) => Point,
    flips: readonly Flip[],
): TourFacts {
    const taken = changes.filter((change): change is TileChange & {from: string} => change.to === flag && change.from !== undefined && change.from !== flag)
    const flipped = flippedTo(flips, flag)
    return {
        took: taken.length,
        byContinent: top(tally(taken.flatMap(({tile}) => continentOf(tile) ?? [])), Infinity),
        tookFrom: top(tally(taken.map(({from}) => from)), MOST_HOLDERS),
        solidity: solidityAfter(flag, changes, after, pointOf),
        flippedByContinent: top(sumBy(flipped.flatMap(({tile, tiles}) => {
            const continent = continentOf(tile)
            return continent === undefined ? [] : [[continent, tiles] as const]
        })), Infinity),
    }
}

// owners: the map as it stands. tiles: how many tiles the map has, ids from 1. fought: what changed hands in the window.
// The flag holding the most of each place that was fought over, the most held first: the world, each continent and each
// big country.
export function placesOf(
    owners: Owners,
    tiles: number,
    groundOf: (tile: number) => string | undefined,
    fought: readonly TileChange[],
    regionOf: RegionOf,
): PlaceFacts[] {
    const sizes = tally(Array.from({length: tiles}, (_, i) => groundOf(i + 1)).flatMap((ground) => ground ?? []))
    const keysOf = memo((ground: string | undefined) => placeKeysOf(ground, sizes, regionOf))

    const grounds = new Map<string, number>()
    const held = new Map<string, Map<string, number>>()
    for (let tile = 1; tile <= tiles; tile++) {
        const owner = owners.get(tile)
        for (const key of keysOf(groundOf(tile))) {
            grounds.set(key, (grounds.get(key) ?? 0) + 1)
            if (owner === undefined) continue
            const there = held.get(key) ?? new Map<string, number>()
            there.set(owner, (there.get(owner) ?? 0) + 1)
            held.set(key, there)
        }
    }

    const tops = [...held].flatMap(([key, there]) => {
        const [first, ...rest] = ranked(there)
        return first === undefined ? [] : [{key, holder: first[0], tiles: first[1], rivals: Object.fromEntries(rest.slice(0, RIVALS))}]
    })
    const holderOf = new Map(tops.map(({key, holder}) => [key, holder]))
    const fights = tally(fought.flatMap(({tile, from, to}) => from === to ? [] : keysOf(groundOf(tile)).filter((key) => {
        const holder = holderOf.get(key)
        return holder !== undefined && (holder === from || holder === to)
    })))

    return tops.flatMap(({key, holder, tiles: count, rivals}): PlaceFacts[] => {
        const changed = fights.get(key) ?? 0
        if (changed === 0) return []
        const ground = grounds.get(key) ?? 0
        return [{...placeOf(key, regionOf), tiles: ground, holder, held: count, share: count / Math.max(1, ground), rivals, changed}]
    }).sort((a, b) => b.held - a.held)
}

export function holdingStoryOf(place: PlaceFacts): Story {
    return {kind: "holding", attacker: place.holder, rival: undefined, victims: Object.keys(place.rivals), place: place.place, share: place.share}
}

// Whether a country's ground counts for a holding of a place: what the headline's share is of.
export function inHolding(place: Place, ground: string | undefined, regionOf: RegionOf): boolean {
    if ("region" in place && place.region === THE_WORLD) return true
    if (ground === undefined) return false
    if ("country" in place) return ground === place.country
    if ("countries" in place) return place.countries.includes(ground)
    return ground !== OUTSIDE_EUROPE && regionOf(ground) === place.region
}

function placeKeysOf(ground: string | undefined, sizes: ReadonlyMap<string, number>, regionOf: RegionOf): string[] {
    if (ground === undefined) return [THE_WORLD]
    const continent = ground === OUTSIDE_EUROPE ? undefined : regionOf(ground)
    return [
        THE_WORLD,
        ...continent === undefined ? [] : [`region:${continent}`],
        ...(sizes.get(ground) ?? 0) >= BIG_COUNTRY ? [`country:${ground}`] : [],
    ]
}

function placeOf(key: string, regionOf: RegionOf): Pick<PlaceFacts, "kind" | "place" | "code" | "continent"> {
    if (key === THE_WORLD) return {kind: "world", place: {region: THE_WORLD}, code: undefined, continent: undefined}
    const [scale, name] = key.split(":")
    return scale === "country"
        ? {kind: "country", place: {country: name}, code: name, continent: regionOf(name)}
        : {kind: "continent", place: {region: name}, code: undefined, continent: name}
}

function memo<T>(compute: (ground: string | undefined) => T): (ground: string | undefined) => T {
    const known = new Map<string | undefined, T>()
    return (ground) => {
        if (!known.has(ground)) known.set(ground, compute(ground))
        return known.get(ground)!
    }
}
