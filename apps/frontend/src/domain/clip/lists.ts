import {BoardScale} from "../boardScale.ts"
import {ranked, tally, TileChange} from "./changes.ts"
import {Owners} from "./findings.ts"
import {dot, meanOf, Point} from "./geometry.ts"
import {readableFlipTiles} from "./look.ts"

// A carousel's lists (docs/video-lab.md, "Photos, not a video"), each ranked, its top first.
export type ListName = "fight-the-most" | "far-from-home" | "biggest-empires"

export const LIST_NAMES: readonly ListName[] = ["fight-the-most", "far-from-home", "biggest-empires"]

// The ranks a carousel tells, between its cover and its last photo.
export const LIST_LENGTH = 5

// fought: the tiles it took from other flags and those they took from it. rival: the flag it fought the most tiles with.
export type Fighting = {flag: string, fought: number, took: number, lost: number, rival: string, rivalTiles: number, country: string}

export type FarLand = {flag: string, country: string, tiles: number, km: number}

export type Empire = {flag: string, tiles: number, share: number, country: string}

export type Lists = {fightTheMost: Fighting[], farFromHome: FarLand[], biggestEmpires: Empire[]}

export type Entry = Fighting | FarLand | Empire

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

// fights: what changed hands in the stretch fight-the-most counts.
export function listsOf(ground: ListGround, fights: readonly TileChange[], scale: BoardScale): Lists {
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
        fightTheMost: fightTheMostOf(fights, ground.groundOf),
        farFromHome: farFromHomeOf(grounds, holdings, ground.pointOf, scale),
        biggestEmpires: biggestEmpiresOf(ground.owners, ground.tiles, holdings),
    }
}

const foughtOf = (fights: readonly TileChange[]) => fights.filter((change): change is TileChange & {from: string, to: string} =>
    change.from !== undefined && change.to !== undefined && change.from !== change.to)

function fightTheMostOf(fights: readonly TileChange[], groundOf: (tile: number) => string | undefined): Fighting[] {
    const fought = foughtOf(fights)
    const took = tally(fought.map(({to}) => to))
    const lost = tally(fought.map(({from}) => from))
    return ranked(tally(fought.flatMap(({from, to}) => [from, to]))).slice(0, LIST_LENGTH).map(([flag, tiles]) => {
        const its = fought.filter(({from, to}) => from === flag || to === flag)
        const [rival] = ranked(tally(its.map(({from, to}) => from === flag ? to : from)))
        const [where] = ranked(tally(its.flatMap(({tile}) => groundOf(tile) ?? [])))
        return {
            flag, fought: tiles, took: took.get(flag) ?? 0, lost: lost.get(flag) ?? 0, rival: rival[0], rivalTiles: rival[1],
            country: where?.[0] ?? "",
        }
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
        case "fight-the-most":
            return lists.fightTheMost
        case "far-from-home":
            return lists.farFromHome
        case "biggest-empires":
            return lists.biggestEmpires
    }
}

export function codeOf(entry: Entry): string {
    return entry.flag
}

// The flag's tiles on the entry's country, or those it fought for there.
export function aimedTilesOf(entry: Entry, ground: ListGround, fights: readonly TileChange[]): number[] {
    const there = (tile: number) => entry.country === "" || ground.groundOf(tile) === entry.country
    if ("fought" in entry) return foughtOf(fights).flatMap(({tile, from, to}) => (from === entry.flag || to === entry.flag) && there(tile) ? [tile] : [])
    return [...ground.owners].flatMap(([tile, owner]) => owner === entry.flag && there(tile) ? [tile] : [])
}
