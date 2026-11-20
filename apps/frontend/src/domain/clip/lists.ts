import {BoardScale} from "../boardScale.ts"
import {ranked, tally, TileChange} from "./changes.ts"
import {bigCountryTiles, Owners} from "./findings.ts"
import {dot, meanOf, Point} from "./geometry.ts"
import {readableFlipTiles} from "./look.ts"

// A carousel's lists (docs/video-lab.md, "Photos, not a video"), each ranked, its top first.
export type ListName = "own-land" | "most-attacked" | "far-from-home" | "biggest-empires"

export const LIST_NAMES: readonly ListName[] = ["own-land", "most-attacked", "far-from-home", "biggest-empires"]

// The ranks a carousel tells, between its cover and its last photo.
export const LIST_LENGTH = 5

export type OwnLand = {country: string, tiles: number, ownShare: number, holder: string, holderShare: number}

export type Attacked = {flag: string, lost: number, takenBy: string, takenByTiles: number, country: string}

export type FarLand = {flag: string, country: string, tiles: number, km: number}

export type Empire = {flag: string, tiles: number, share: number, country: string}

export type Lists = {ownLand: OwnLand[], mostAttacked: Attacked[], farFromHome: FarLand[], biggestEmpires: Empire[]}

export type Entry = OwnLand | Attacked | FarLand | Empire

const EARTH_KM = 6371

const KM_STEP = 100

// A country's ground is read at this many of its tiles at most when the distance to it is measured.
const HOME_POINTS = 2000

export type ListGround = {
    owners: Owners
    tiles: number
    groundOf: (tile: number) => string | undefined
    pointOf: (tile: number) => Point
}

// attacks: what changed hands in the stretch most-attacked counts.
export function listsOf(ground: ListGround, attacks: readonly TileChange[], scale: BoardScale): Lists {
    const grounds = new Map<string, number[]>()
    for (let tile = 1; tile <= ground.tiles; tile++) {
        const country = ground.groundOf(tile)
        if (country === undefined) continue
        const list = grounds.get(country)
        if (list) list.push(tile)
        else grounds.set(country, [tile])
    }
    const holdings = new Map<string, Map<string, number[]>>()
    for (const [tile, flag] of ground.owners) {
        const country = ground.groundOf(tile)
        if (country === undefined) continue
        const held = holdings.get(flag) ?? new Map<string, number[]>()
        const there = held.get(country)
        if (there) there.push(tile)
        else held.set(country, [tile])
        holdings.set(flag, held)
    }
    return {
        ownLand: ownLandOf(grounds, ground.owners, scale),
        mostAttacked: mostAttackedOf(attacks, ground.groundOf),
        farFromHome: farFromHomeOf(grounds, holdings, ground.pointOf, scale),
        biggestEmpires: biggestEmpiresOf(ground.owners, ground.tiles, holdings),
    }
}

// A country nobody holds any of has no holder to tell: it is left out.
function ownLandOf(grounds: ReadonlyMap<string, readonly number[]>, owners: Owners, scale: BoardScale): OwnLand[] {
    const least = bigCountryTiles(scale)
    return [...grounds].flatMap(([country, tiles]): OwnLand[] => {
        if (tiles.length < least) return []
        const held = tally(tiles.flatMap((tile) => owners.get(tile) ?? []))
        const [top] = ranked(held)
        if (top === undefined) return []
        return [{country, tiles: tiles.length, ownShare: (held.get(country) ?? 0) / tiles.length, holder: top[0], holderShare: top[1] / tiles.length}]
    }).sort((a, b) => a.ownShare - b.ownShare || b.holderShare - a.holderShare || b.tiles - a.tiles || a.country.localeCompare(b.country))
        .slice(0, LIST_LENGTH)
}

function mostAttackedOf(attacks: readonly TileChange[], groundOf: (tile: number) => string | undefined): Attacked[] {
    const taken = attacks.filter((change): change is TileChange & {from: string, to: string} =>
        change.from !== undefined && change.to !== undefined && change.from !== change.to)
    return ranked(tally(taken.map(({from}) => from))).slice(0, LIST_LENGTH).map(([flag, lost]) => {
        const fromIt = taken.filter(({from}) => from === flag)
        const [by] = ranked(tally(fromIt.map(({to}) => to)))
        const [where] = ranked(tally(fromIt.flatMap(({tile}) => groundOf(tile) ?? [])))
        return {flag, lost, takenBy: by[0], takenByTiles: by[1], country: where?.[0] ?? ""}
    })
}

// A holding the size of a painted flag read from far counts half its distance, a speck nearly none.
function farFromHomeOf(
    grounds: ReadonlyMap<string, readonly number[]>,
    holdings: ReadonlyMap<string, ReadonlyMap<string, readonly number[]>>,
    pointOf: (tile: number) => Point,
    scale: BoardScale,
): FarLand[] {
    const half = readableFlipTiles(scale)
    const far = [...holdings].flatMap(([flag, held]) => {
        const home = grounds.get(flag)
        if (home === undefined) return []
        const step = Math.max(1, Math.floor(home.length / HOME_POINTS))
        const homePoints = home.filter((_, i) => i % step === 0).map(pointOf)
        let best: {land: FarLand, score: number} | undefined
        for (const [country, tiles] of held) {
            const centre = meanOf(tiles.map(pointOf))
            if (country === flag || centre === undefined) continue
            const nearest = Math.max(...homePoints.map((point) => dot(point, centre)))
            const km = Math.acos(Math.min(1, Math.max(-1, nearest))) * EARTH_KM
            const score = km * tiles.length / (tiles.length + half)
            if (best === undefined || score > best.score) {
                best = {land: {flag, country, tiles: tiles.length, km: Math.round(km / KM_STEP) * KM_STEP}, score}
            }
        }
        return best === undefined ? [] : [best]
    })
    return far.sort((a, b) => b.score - a.score || a.land.flag.localeCompare(b.land.flag)).slice(0, LIST_LENGTH).map(({land}) => land)
}

function biggestEmpiresOf(owners: Owners, tiles: number, holdings: ReadonlyMap<string, ReadonlyMap<string, readonly number[]>>): Empire[] {
    return ranked(tally(owners.values())).slice(0, LIST_LENGTH).map(([flag, held]) => {
        const [most] = ranked(new Map([...holdings.get(flag) ?? []].map(([country, there]) => [country, there.length])))
        return {flag, tiles: held, share: held / Math.max(1, tiles), country: most?.[0] ?? ""}
    })
}

export function entriesOf(lists: Lists, list: ListName): readonly Entry[] {
    switch (list) {
        case "own-land":
            return lists.ownLand
        case "most-attacked":
            return lists.mostAttacked
        case "far-from-home":
            return lists.farFromHome
        case "biggest-empires":
            return lists.biggestEmpires
    }
}

// The country of own-land, the flag of the others: what a rank names.
export function codeOf(entry: Entry): string {
    return "flag" in entry ? entry.flag : entry.country
}

// Own-land's country itself; the flag's tiles on the entry's country, or those taken from it there.
export function aimedTilesOf(entry: Entry, ground: ListGround, attacks: readonly TileChange[]): number[] {
    if (!("flag" in entry)) {
        return Array.from({length: ground.tiles}, (_, i) => i + 1).filter((tile) => ground.groundOf(tile) === entry.country)
    }
    const there = (tile: number) => entry.country === "" || ground.groundOf(tile) === entry.country
    if ("lost" in entry) return attacks.flatMap(({tile, from, to}) => from === entry.flag && to !== undefined && to !== from && there(tile) ? [tile] : [])
    return [...ground.owners].flatMap(([tile, owner]) => owner === entry.flag && there(tile) ? [tile] : [])
}
