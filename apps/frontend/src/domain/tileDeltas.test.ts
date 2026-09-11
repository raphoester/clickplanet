import {describe, expect, it} from "vitest"
import {Countries} from "./countries.ts"
import {
    DELTA_HOLD_MS,
    expireBadges,
    NO_TILE_DELTAS,
    signed,
    takeInChanges,
    TileDeltas,
    tileCounts,
} from "./tileDeltas.ts"

const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

const counts = (pairs: Record<string, number>) => new Map(Object.entries(pairs))

const nets = (badges: TileDeltas) =>
    Object.fromEntries([...badges].map(([code, badge]) => [code, badge.net]))

describe("tileCounts", () => {
    it("reads a board as tiles per country", () => {
        expect(tileCounts([entry("fr", 12), entry("jp", 3)]))
            .toEqual(counts({fr: 12, jp: 3}))
    })

    it("reads an empty board as nobody holding anything", () => {
        expect(tileCounts([])).toEqual(new Map())
    })
})

describe("takeInChanges", () => {
    it("badges a country that wins a tile", () => {
        const badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        expect(nets(badges)).toEqual({fr: 1})
    })

    it("badges a country that loses one", () => {
        const badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 10}), 0)
        expect(nets(badges)).toEqual({fr: -2})
    })

    it("counts a run of wins into one badge instead of three", () => {
        let badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        badges = takeInChanges(badges, counts({fr: 13}), counts({fr: 14}), 400)
        badges = takeInChanges(badges, counts({fr: 14}), counts({fr: 15}), 900)

        expect(nets(badges)).toEqual({fr: 3})
        expect(badges.get("fr")!.at).toBe(900)
    })

    it("replays the badge on every change it takes in", () => {
        const first = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        const second = takeInChanges(first, counts({fr: 13}), counts({fr: 14}), 400)

        expect(second.get("fr")!.beat).toBeGreaterThan(first.get("fr")!.beat)
    })

    it("holds the two sides of a stolen tile apart", () => {
        const badges = takeInChanges(
            NO_TILE_DELTAS, counts({fr: 12, jp: 4}), counts({fr: 13, jp: 3}), 0)

        expect(nets(badges)).toEqual({fr: 1, jp: -1})
    })

    it("drops a badge whose country ends up where it started", () => {
        const won = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        const takenBack = takeInChanges(won, counts({fr: 13}), counts({fr: 12}), 200)

        expect(takenBack.has("fr")).toBe(false)
    })

    it("counts a country onto the board from nothing, and off it again", () => {
        const arrived = takeInChanges(NO_TILE_DELTAS, counts({}), counts({jp: 2}), 0)
        expect(nets(arrived)).toEqual({jp: 2})

        const gone = takeInChanges(NO_TILE_DELTAS, counts({jp: 2}), counts({}), 0)
        expect(nets(gone)).toEqual({jp: -2})
    })

    it("leaves a board that did not move alone", () => {
        const badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 12}), 0)
        expect(badges.size).toBe(0)
    })

    it("starts a fresh badge once the last one has been retired", () => {
        const first = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        const retired = expireBadges(first, DELTA_HOLD_MS + 1)
        const later = takeInChanges(
            retired, counts({fr: 13}), counts({fr: 14}), DELTA_HOLD_MS + 1)

        expect(nets(later)).toEqual({fr: 1})
    })

    it("hands back the badges it was given when the board did not move", () => {
        const first = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        expect(takeInChanges(first, counts({fr: 13}), counts({fr: 13}), 100)).toBe(first)
    })

    it("does not touch the badges it was handed", () => {
        const first = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 0)
        takeInChanges(first, counts({fr: 13}), counts({fr: 20}), 100)

        expect(nets(first)).toEqual({fr: 1})
    })
})

describe("expireBadges", () => {
    it("keeps a badge for its couple of seconds, then drops it", () => {
        const badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 1_000)

        expect(expireBadges(badges, 1_000 + DELTA_HOLD_MS - 1).has("fr")).toBe(true)
        expect(expireBadges(badges, 1_000 + DELTA_HOLD_MS).has("fr")).toBe(false)
    })

    it("hands back the badges it was given when none of them is due", () => {
        const badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 12}), counts({fr: 13}), 1_000)
        expect(expireBadges(badges, 1_500)).toBe(badges)
    })

    it("drops only the badge that is due", () => {
        let badges = takeInChanges(NO_TILE_DELTAS, counts({fr: 1}), counts({fr: 2}), 0)
        badges = takeInChanges(badges, counts({jp: 1}), counts({jp: 2}), 1_000)

        expect([...expireBadges(badges, DELTA_HOLD_MS + 1).keys()]).toEqual(["jp"])
    })
})

describe("signed", () => {
    it("spells a gain with its plus, and a loss with its minus", () => {
        expect(signed(3)).toBe("+3")
        expect(signed(-3)).toBe("-3")
    })
})
