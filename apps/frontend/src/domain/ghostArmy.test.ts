import {describe, expect, it} from "vitest"
import {GhostArmy, Ground, Sight} from "./ghostArmy.ts"
import {GhostTake, Ghosts} from "./ghosts.ts"

// A square of tiles around (1, 0, 0), 0.01 apart, each touching the four beside it; tile = row * SIDE + column + 1.
const SIDE = 10

const STEP = 0.01

const tileAt = (row: number, column: number) => row * SIDE + column + 1

const columnOf = (tile: number) => (tile - 1) % SIDE

function positionsOfSquare(): Float32Array {
    const positions = new Float32Array(SIDE * SIDE * 3)
    for (let row = 0; row < SIDE; row++) {
        for (let column = 0; column < SIDE; column++) {
            const y = (column - 4.5) * STEP, z = (row - 4.5) * STEP
            const length = Math.hypot(1, y, z)
            positions.set([1 / length, y / length, z / length], (tileAt(row, column) - 1) * 3)
        }
    }
    return positions
}

function neighboursOf(tile: number): number[] {
    const row = Math.floor((tile - 1) / SIDE), column = columnOf(tile)
    return [[row - 1, column], [row + 1, column], [row, column - 1], [row, column + 1]]
        .filter(([r, c]) => r >= 0 && r < SIDE && c >= 0 && c < SIDE)
        .map(([r, c]) => tileAt(r, c))
}

type World = {
    owners: Map<number, string>
    shields?: Map<number, number>
    grounds?: Map<number, string>
    counts: Map<string, number>
}

function groundOf({owners, shields, grounds, counts}: World): Ground {
    return {
        positions: positionsOfSquare(),
        neighboursOf,
        ownerOf: (tile) => owners.get(tile),
        shieldsOf: (tile) => shields?.get(tile) ?? 0,
        groundOf: (tile) => grounds?.get(tile),
        countOf: (flag) => counts.get(flag) ?? 0,
    }
}

function split(left: string, right: string): Map<number, string> {
    const owners = new Map<number, string>()
    for (let tile = 1; tile <= SIDE * SIDE; tile++) owners.set(tile, columnOf(tile) < SIDE / 2 ? left : right)
    return owners
}

const steady = () => 0

const SEES_ALL: Sight = {flag: "bg", centre: {x: 1, y: 0, z: 0}, least: 0.99}

function march(army: GhostArmy, ghosts: Ghosts, from: number, to: number, sight = SEES_ALL): (GhostTake & {at: number})[] {
    const takes: (GhostTake & {at: number})[] = []
    for (let now = from; now <= to; now += 50) {
        const take = army.march(now, sight)
        if (!take) continue
        ghosts.take(take)
        takes.push({...take, at: now})
    }
    return takes
}

describe("GhostArmy", () => {
    it("takes nothing at once, then lands on the player's front under the flag next door", () => {
        const owners = split("bg", "fr")
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners, counts: new Map([["bg", 50]])}), ghosts, steady, 0)

        expect(army.march(0, SEES_ALL)).toBeUndefined()
        const take = army.march(60_000, SEES_ALL)

        expect(take && owners.get(take.tile)).toBe("bg")
        expect(take && columnOf(take.tile)).toBe(4)
        expect(take?.flag).toBe("fr")
        expect(take && take.until - 60_000).toBeGreaterThanOrEqual(60_000)
    })

    it("grows from the tiles it took", () => {
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners: split("bg", "fr"), counts: new Map([["bg", 50]])}), ghosts, steady, 0)

        const takes = march(army, ghosts, 0, 5_000)

        expect(takes.length).toBeGreaterThan(1)
        takes.slice(1).forEach((take, i) => {
            expect(takes.slice(0, i + 1).some(({tile}) => neighboursOf(take.tile).includes(tile))).toBe(true)
        })
    })

    it("taps a few tiles, then pauses", () => {
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners: split("bg", "fr"), counts: new Map([["bg", 50]])}), ghosts, steady, 0)

        const times = march(army, ghosts, 0, 25_000).map(({at}) => at)
        const gaps = times.slice(1).map((time, i) => time - times[i])

        expect(gaps.length).toBeGreaterThan(3)
        for (const gap of gaps) expect(gap < 1_200 || gap >= 6_000).toBe(true)
        expect(gaps.some((gap) => gap >= 6_000)).toBe(true)
    })

    it("holds no more than a quarter of the player's tiles", () => {
        const owners = split("bg", "fr")
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners, counts: new Map([["bg", 8]])}), ghosts, steady, 0)

        const takes = march(army, ghosts, 0, 120_000)

        expect(takes.filter(({tile}) => owners.get(tile) === "bg")).toHaveLength(2)
    })

    it("takes no free tile, no shielded tile and no tile out of sight", () => {
        const owners = new Map([[tileAt(0, 0), "bg"], [tileAt(0, 1), "fr"], [tileAt(5, 4), "bg"], [tileAt(5, 5), "fr"]])
        const shields = new Map([[tileAt(5, 4), 1], [tileAt(5, 5), 2]])
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners, shields, counts: new Map([["bg", 40]])}), ghosts, steady, 0)
        const middle = {...SEES_ALL, least: Math.cos(0.03)}

        expect(march(army, ghosts, 0, 60_000, middle)).toEqual([])
    })

    it("plays the nearest other country by its ground when no flag is near", () => {
        const owners = new Map<number, string>()
        const grounds = new Map<number, string>()
        for (let tile = 1; tile <= SIDE * SIDE; tile++) {
            if (columnOf(tile) >= 3 && columnOf(tile) <= 5) owners.set(tile, "bg")
            grounds.set(tile, columnOf(tile) >= 8 ? "ro" : "bg")
        }
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners, grounds, counts: new Map([["bg", 30]])}), ghosts, steady, 0)

        expect(army.march(60_000, SEES_ALL)?.flag).toBe("ro")
    })

    it("never plays the player's flag, even when it is the only flag next door", () => {
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners: split("bg", "fr"), counts: new Map()}), ghosts, steady, 0)

        expect(march(army, ghosts, 0, 60_000)).toEqual([])
    })

    it("stops a raid as soon as the player takes back one of its tiles", () => {
        const ghosts = new Ghosts([], 0)
        const army = new GhostArmy(groundOf({owners: split("bg", "fr"), counts: new Map([["bg", 50]])}), ghosts, steady, 0)
        const [first, second] = march(army, ghosts, 0, 3_350)

        ghosts.release([first.tile])

        expect(second).toBeDefined()
        expect(march(army, ghosts, 3_400, 9_000)).toEqual([])
    })
})
