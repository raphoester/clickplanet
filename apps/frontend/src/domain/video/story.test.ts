import {describe, expect, it} from "vitest"
import {alliesOf, castOf, Ground, headlineOf, movedShare, percentOf, RecentFrame, RecentHistory, storyOf, VideoStory} from "./story.ts"

// Tiles 1-100 are Germany, 101-200 Poland, 201 Monaco.
const ground: Ground = {
    tiles: 201,
    groundOf: (tile) => tile <= 100 ? "de" : tile <= 200 ? "pl" : tile === 201 ? "mc" : undefined,
}

const range = (from: number, to: number) => Array.from({length: to - from + 1}, (_, i) => from + i)

function frame(at: number, changes: [tile: number, owner: string, account?: string][]): RecentFrame {
    return {
        at,
        tiles: Uint32Array.from(changes.map(([tile]) => tile)),
        owners: changes.map(([, owner]) => owner),
        changedAt: new Float64Array(changes.length).fill(at),
        accounts: changes.map(([, , account]) => account ?? ""),
    }
}

function historyOf(...frames: RecentFrame[]): RecentHistory {
    return {frames, hits: new Map()}
}

const naming = {nameOf: (code: string) => ({pl: "Poland", de: "Germany", gb: "UK", mc: "Monaco"})[code] ?? code, said: (name: string) => name === "UK" ? `the ${name}` : name}

describe("the video's story", () => {
    it("is the real country where the flag gained the most tiles", () => {
        const history = historyOf(
            frame(0, range(101, 200).map((tile) => [tile, "pl"])),
            frame(1, range(1, 40).map((tile) => [tile, "pl", "kutza"])),
            frame(2, [[201, "pl", "kutza"]]),
        )

        expect(storyOf(history, ground, "pl")).toEqual({flag: "pl", country: "de", kind: "gain", tiles: 100, start: 0, end: 40})
    })

    it("is a loss where the flag lost more tiles than it gained anywhere", () => {
        const history = historyOf(
            frame(0, range(101, 200).map((tile) => [tile, "pl"])),
            frame(1, [...range(101, 130).map((tile): [number, string] => [tile, "de"]), ...range(1, 10).map((tile): [number, string] => [tile, "pl"])]),
        )

        expect(storyOf(history, ground, "pl")).toMatchObject({country: "pl", kind: "loss", start: 100, end: 70})
    })

    it("is a quiet day under a hundredth of a country, told where the flag holds the most by its share", () => {
        // Tiles 1-1000 are Russia, 1001-1100 Poland, 1101 Monaco.
        const wide: Ground = {tiles: 1101, groundOf: (tile) => tile <= 1000 ? "ru" : tile <= 1100 ? "pl" : "mc"}
        const history = historyOf(
            frame(0, [...range(1001, 1062).map((tile): [number, string] => [tile, "pl"]), [1101, "pl"], ...range(1, 9).map((tile): [number, string] => [tile, "pl"])]),
            frame(1, range(1, 9).map((tile) => [tile, "ru"])),
        )

        expect(storyOf(history, wide, "pl")).toMatchObject({country: "pl", kind: "quiet", start: 62, end: 62})
    })

    it("counts a tile a bomb left empty as lost, and nothing for a flag with no land", () => {
        const history = historyOf(frame(0, range(1, 20).map((tile) => [tile, "pl"])), frame(1, range(1, 20).map((tile) => [tile, "", "bomber"])))

        expect(storyOf(history, ground, "pl")).toMatchObject({country: "de", kind: "loss", start: 20, end: 0})
        expect(storyOf(historyOf(frame(0, [])), ground, "pl")).toBeUndefined()
        expect(storyOf(historyOf(), ground, "pl")).toBeUndefined()
    })

    it("says how much of its country moved", () => {
        expect(movedShare({flag: "pl", country: "de", kind: "loss", tiles: 200, start: 50, end: 10})).toBe(0.2)
    })
})

describe("the video's headline", () => {
    const story = (kind: VideoStory["kind"], start: number, end: number): VideoStory => ({flag: "pl", country: "de", kind, tiles: 100, start, end})

    it("is one fixed sentence for each kind of day, filled with its facts", () => {
        expect(headlineOf(story("gain", 0, 40), naming)).toBe("POLAND TOOK 40% OF GERMANY")
        expect(headlineOf(story("loss", 60, 10), naming)).toBe("POLAND IS LOSING GERMANY. HELP")
        expect(headlineOf(story("quiet", 62, 62), naming)).toBe("POLAND HOLDS 62% OF GERMANY")
    })

    it("says a name with its article", () => {
        expect(headlineOf({...story("gain", 0, 40), flag: "gb"}, naming)).toBe("THE UK TOOK 40% OF GERMANY")
    })

    it("never rounds a share up, and never says 0% of something", () => {
        expect(percentOf(399, 1000)).toBe(39)
        expect(percentOf(1, 1000)).toBe(1)
        expect(percentOf(0, 1000)).toBe(0)
    })
})

describe("the video's allies", () => {
    const history = historyOf(
        frame(0, range(1, 10).map((tile) => [tile, "de"])),
        frame(1, [[1, "pl", "ada"], [2, "pl", "bob"], [3, "pl", "bob"], [4, "de", "hans"], [101, "pl", "cy"], [5, "pl", "maker"], [6, "pl", ""]]),
        frame(2, [[7, "pl", "ada"], [8, "pl", "ada"], [9, "pl", "dee"]]),
    )
    const story: VideoStory = {flag: "pl", country: "de", kind: "gain", tiles: 100, start: 0, end: 8}

    it("are the accounts that won the flag the most of the story's country, never one that took from it, nor the maker", () => {
        expect(alliesOf(history, ground, story, "maker")).toEqual(["ada", "bob", "dee"])
    })

    it("name only who the clips let a video show, three at most, and the maker by its shown name", () => {
        const shown = new Map([["ada", "Ada"], ["dee", "Dee"], ["maker", "kutza"], ["eve", "Eve"], ["fay", "Fay"]])

        expect(castOf("maker", ["ada", "bob", "dee", "eve", "fay"], shown)).toEqual({maker: "kutza", allies: ["Ada", "Dee", "Eve"]})
        expect(castOf(undefined, [], shown)).toEqual({maker: undefined, allies: []})
        expect(castOf("bob", [], shown)).toEqual({maker: undefined, allies: []})
    })
})
