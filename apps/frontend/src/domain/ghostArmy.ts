import {Point} from "./clip/geometry.ts"
import {GhostTake, Ghosts} from "./ghosts.ts"

export type Ground = {
    // positions holds x, y, z per tile, tile id - 1 first.
    positions: ArrayLike<number>
    neighboursOf(tile: number): readonly number[]
    // The server's flag, under the ghosts.
    ownerOf(tile: number): string | undefined
    shieldsOf(tile: number): number
    // The country whose ground the tile is on, when its flag can be painted.
    groundOf(tile: number): string | undefined
    countOf(flag: string): number
}

// A tile is in sight when the dot of its direction with the centre is at least least.
export type Sight = {flag: string, centre: Point, least: number}

type Span = readonly [number, number]

const FIRST_RAID_MS: Span = [3_000, 8_000]

const TAP_MS: Span = [350, 1_100]

const PAUSE_MS: Span = [6_000, 16_000]

const LOOK_AGAIN_MS = 2_000

const RAID_TAKES: Span = [3, 8]

const LIFE_MS: Span = [60_000, 180_000]

const MOST_GHOSTS = 40

const MOST_FROM_FLAG = 12

const SHARE_FROM_FLAG = 0.25

const OWN_FRONT_FIRST = 0.75

const LANDINGS_TRIED = 5

const RIVAL_SEARCH = 4_000

type Raid = {flag: string, tiles: number[], left: number}

// Raids on what the player sees, its own front first: a few quick taps from one rival, then a pause.
export class GhostArmy {
    private raid: Raid | undefined
    private nextAt: number

    constructor(
        private readonly ground: Ground,
        private readonly ghosts: Ghosts,
        private readonly random: () => number,
        now: number,
    ) {
        this.nextAt = now + this.within(FIRST_RAID_MS)
    }

    public march(now: number, sight: Sight): GhostTake | undefined {
        if (now < this.nextAt) return undefined
        const full = this.ghosts.size >= MOST_GHOSTS
        if (this.raid) {
            const tile = full ? undefined : this.push(this.raid, sight)
            if (tile === undefined) {
                this.rest(now)
                return undefined
            }
            return this.taken(this.raid, tile, now)
        }

        const landing = full ? undefined : this.landing(sight)
        if (!landing) {
            this.nextAt = now + LOOK_AGAIN_MS
            return undefined
        }
        this.raid = {flag: landing.flag, tiles: [], left: Math.round(this.within(RAID_TAKES))}
        return this.taken(this.raid, landing.tile, now)
    }

    private taken(raid: Raid, tile: number, now: number): GhostTake {
        raid.tiles.push(tile)
        raid.left--
        if (raid.left > 0) this.nextAt = now + this.within(TAP_MS)
        else this.rest(now)
        return {tile, flag: raid.flag, until: now + this.within(LIFE_MS)}
    }

    private rest(now: number) {
        this.raid = undefined
        this.nextAt = now + this.within(PAUSE_MS)
    }

    private landing(sight: Sight): {tile: number, flag: string} | undefined {
        const fromFlag = this.fromFlag(sight.flag)
        const own: number[] = []
        const others: number[] = []
        for (let tile = 1; tile * 3 <= this.ground.positions.length; tile++) {
            if (!this.inSight(tile, sight)) continue
            const owner = this.open(tile)
            if (owner === undefined || !this.border(tile, owner)) continue
            if (owner !== sight.flag) others.push(tile)
            else if (fromFlag) own.push(tile)
        }

        const pool = own.length > 0 && (others.length === 0 || this.random() < OWN_FRONT_FIRST) ? own : others
        for (let tried = 0; tried < LANDINGS_TRIED && pool.length > 0; tried++) {
            const at = Math.floor(this.random() * pool.length)
            const tile = pool[at]
            const flag = this.rivalOf(tile, sight.flag)
            if (flag !== undefined) return {tile, flag}
            pool[at] = pool[pool.length - 1]
            pool.pop()
        }
        return undefined
    }

    private push(raid: Raid, sight: Sight): number | undefined {
        if (raid.flag === sight.flag || raid.tiles.some((tile) => this.ghosts.flagOf(tile) !== raid.flag)) return undefined
        const fromFlag = this.fromFlag(sight.flag)
        const seen = new Set<number>()
        let best: number | undefined
        let bestScore = -Infinity
        for (const held of raid.tiles) {
            for (const tile of this.ground.neighboursOf(held)) {
                if (seen.has(tile)) continue
                seen.add(tile)
                if (!this.inSight(tile, sight)) continue
                const owner = this.open(tile)
                if (owner === undefined || owner === raid.flag || (owner === sight.flag && !fromFlag)) continue
                const score = (owner === sight.flag ? 10 : 0) + this.around(tile, raid.flag) + this.random()
                if (score > bestScore) {
                    bestScore = score
                    best = tile
                }
            }
        }
        return best
    }

    private rivalOf(start: number, flag: string): string | undefined {
        const owner = this.ground.ownerOf(start)
        const rival = (code: string | undefined): code is string => code !== undefined && code !== owner && code !== flag

        const next = mostCommon(this.ground.neighboursOf(start).map((tile) => this.shownOwnerOf(tile)).filter(rival))
        if (next !== undefined) return next

        const queue = [start]
        const seen = new Set(queue)
        for (let i = 0; i < queue.length && i < RIVAL_SEARCH; i++) {
            const tile = queue[i]
            const shown = this.shownOwnerOf(tile)
            if (rival(shown)) return shown
            const ground = this.ground.groundOf(tile)
            if (rival(ground)) return ground
            for (const near of this.ground.neighboursOf(tile)) {
                if (seen.has(near)) continue
                seen.add(near)
                queue.push(near)
            }
        }
        return undefined
    }

    private open(tile: number): string | undefined {
        if (this.ghosts.flagOf(tile) !== undefined || this.ground.shieldsOf(tile) > 0) return undefined
        return this.ground.ownerOf(tile)
    }

    private border(tile: number, owner: string): boolean {
        return this.ground.neighboursOf(tile).some((next) => this.shownOwnerOf(next) !== owner)
    }

    private around(tile: number, flag: string): number {
        return this.ground.neighboursOf(tile).filter((next) => this.shownOwnerOf(next) === flag).length
    }

    private fromFlag(flag: string): boolean {
        const most = Math.min(MOST_FROM_FLAG, Math.floor(this.ground.countOf(flag) * SHARE_FROM_FLAG))
        return this.ghosts.heldFrom(flag, (tile) => this.ground.ownerOf(tile)) < most
    }

    private shownOwnerOf(tile: number): string | undefined {
        return this.ghosts.flagOf(tile) ?? this.ground.ownerOf(tile)
    }

    private inSight(tile: number, {centre, least}: Sight): boolean {
        const at = (tile - 1) * 3
        const {positions} = this.ground
        return positions[at] * centre.x + positions[at + 1] * centre.y + positions[at + 2] * centre.z >= least
    }

    private within([least, most]: Span): number {
        return least + this.random() * (most - least)
    }
}

function mostCommon(codes: readonly string[]): string | undefined {
    const tally = new Map<string, number>()
    let common: string | undefined
    for (const code of codes) {
        const count = (tally.get(code) ?? 0) + 1
        tally.set(code, count)
        if (common === undefined || count > (tally.get(common) ?? 0)) common = code
    }
    return common
}
