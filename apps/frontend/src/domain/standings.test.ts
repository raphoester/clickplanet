import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {MySeason, Standing} from "../backends/standings.ts"
import {boardWith, liveSeason, NO_TAKES, STANDINGS_SHOWN, takenCount, takesSince, withTake} from "./standings.ts"

const takes = (entries: Record<string, number>) => new Map(Object.entries(entries))

const standing = (rank: number, name: string, tiles: number): Standing =>
    ({rank, name, color: NameColor.UNSPECIFIED, countryCode: "fr", tiles})

const lines = (board: {listed: Standing[], below?: Standing}) => ({
    listed: board.listed.map((s) => `${s.rank} ${s.name} ${s.tiles}`),
    below: board.below && `${board.below.rank} ${board.below.name} ${board.below.tiles}`,
})

const TOP: Standing[] = Array.from({length: STANDINGS_SHOWN}, (_, i) => standing(i + 1, `p${i + 1}`, 1_000 - i * 100))

describe("takes", () => {
    it("counts one more take of a flag, and leaves the takes it was given alone", () => {
        const before = takes({fr: 2})
        const after = withTake(before, "fr")

        expect(after).toEqual(takes({fr: 3}))
        expect(withTake(after, "de")).toEqual(takes({fr: 3, de: 1}))
        expect(before).toEqual(takes({fr: 2}))
    })

    it("says what was taken since an earlier count, flag by flag", () => {
        expect(takesSince(takes({fr: 2}), takes({fr: 5, de: 1}))).toEqual(takes({fr: 3, de: 1}))
        expect(takesSince(takes({fr: 2}), takes({fr: 2}))).toEqual(NO_TAKES)
    })

    it("adds every flag's takes together", () => {
        expect(takenCount(takes({fr: 3, de: 2}))).toBe(5)
        expect(takenCount(NO_TAKES)).toBe(0)
    })
})

describe("liveSeason", () => {
    const read: MySeason = {countryCode: "fr", tiles: 40, rank: 12}

    it("adds the tiles taken since the read for the flag of the line", () => {
        expect(liveSeason(read, takes({fr: 2}))).toEqual({...read, tiles: 42})
    })

    it("adds nothing for another flag: only the server knows if it became the main one", () => {
        expect(liveSeason(read, takes({de: 50}))).toBe(read)
    })

    it("counts the takes for a country's board into it, though another flag is the main one", () => {
        expect(liveSeason({countryCode: "fr", tiles: 0}, takes({fr: 3, bg: 70}))).toEqual({countryCode: "fr", tiles: 3})
    })

    it("gives a player with no tiles yet the flag of its first takes, as the server will", () => {
        expect(liveSeason({tiles: 0}, takes({de: 2, fr: 1}))).toEqual({countryCode: "de", tiles: 2})
        expect(liveSeason({tiles: 0}, takes({de: 1, fr: 1}))).toEqual({countryCode: "de", tiles: 1})
        expect(liveSeason({tiles: 0}, takes({de: 1, fr: 3}))).toEqual({countryCode: "fr", tiles: 3})
    })

    it("hands back the read while nothing was taken since", () => {
        expect(liveSeason(read, NO_TAKES)).toBe(read)
        expect(liveSeason({tiles: 0}, NO_TAKES)).toEqual({tiles: 0})
    })
})

describe("boardWith", () => {
    it("lists the standings as they are without the caller", () => {
        expect(lines(boardWith(TOP.slice(0, 2), undefined))).toEqual({listed: ["1 p1 1000", "2 p2 900"], below: undefined})
    })

    it("draws the caller's own row with its own count, where that count puts it", () => {
        const board = boardWith(TOP.slice(0, 3), standing(3, "p3", 950))

        expect(lines(board)).toEqual({listed: ["1 p1 1000", "2 p3 950", "3 p2 900"], below: undefined})
    })

    it("lets the caller share the rank of a tie, after the others", () => {
        const board = boardWith(TOP.slice(0, 3), standing(3, "p3", 900))

        expect(lines(board).listed).toEqual(["1 p1 1000", "2 p2 900", "2 p3 900"])
    })

    it("adds a caller missing from a short list to it", () => {
        expect(lines(boardWith(TOP.slice(0, 2), standing(9, "me", 950))).listed).toEqual(["1 p1 1000", "2 me 950", "3 p2 900"])
    })

    it("lets a caller who passes the last of the top in, and the last out", () => {
        const board = boardWith(TOP, standing(14, "me", 150))

        expect(lines(board).listed).toHaveLength(STANDINGS_SHOWN)
        expect(lines(board).listed.slice(-2)).toEqual(["9 p9 200", "10 me 150"])
        expect(board.below).toBeUndefined()
    })

    it("keeps a caller below the top under it, at the rank the server gave", () => {
        const board = boardWith(TOP, standing(14, "me", 50))

        expect(lines(board)).toEqual({listed: TOP.map((s) => `${s.rank} ${s.name} ${s.tiles}`), below: "14 me 50"})
    })
})
