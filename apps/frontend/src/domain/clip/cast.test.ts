import {describe, expect, it} from "vitest"
import {Cast, castWithin, namedSidesOf, Take, topFightersOf} from "./cast.ts"

const take = (fighter: number, flag: string, tile: number, at = 0): Take => ({at, tile, flag, fighter})

const cast: Cast = {
    id: "cast-1",
    names: new Map([[1, "BraveFox42"], [2, "StormWolf"], [3, "ŁukaszPL"], [4, "مقاتل_7"], [5, "Toto"]]),
    takes: [
        take(1, "fr", 1), take(1, "fr", 2), take(1, "fr", 3), take(1, "fr", 4),
        take(2, "tr", 5), take(2, "tr", 6), take(2, "tr", 7),
        take(3, "fr", 8), take(3, "fr", 9), take(3, "tr", 10),
        take(4, "fr", 11), take(4, "fr", 12),
        take(5, "de", 13), take(5, "fr", 14),
    ],
}

describe("the cast within a stretch of time", () => {
    it("keeps the takes of the stretch, both ends in", () => {
        const timed: Cast = {...cast, takes: [take(1, "fr", 1, 10), take(1, "fr", 2, 20), take(1, "fr", 3, 30)]}

        expect(castWithin(timed, 20, 30).takes.map(({tile}) => tile)).toEqual([2, 3])
    })
})

describe("the top fighters of what is counted", () => {
    it("are the players who took the most of it, most first, by number on a tie, three at most", () => {
        expect(topFightersOf(cast, cast.takes.filter(({flag}) => flag === "fr"))).toEqual([
            {number: 1, name: "BraveFox42", took: 4},
            {number: 3, name: "ŁukaszPL", took: 2},
            {number: 4, name: "مقاتل_7", took: 2},
        ])
    })

    it("are nobody when nobody of the cast took any of it", () => {
        expect(topFightersOf(cast, [])).toEqual([])
    })
})

describe("the players a clip names", () => {
    const sameFlag = (flag: string) => flag

    it("are under the side each took the most for, the sides in their order and the most first on each", () => {
        expect(namedSidesOf(cast, [2, 3, 1], cast.takes, ["fr", "tr"], sameFlag)).toEqual([
            {flag: "fr", fighters: [{number: 1, name: "BraveFox42", took: 4}, {number: 3, name: "ŁukaszPL", took: 2}]},
            {flag: "tr", fighters: [{number: 2, name: "StormWolf", took: 3}]},
        ])
    })

    it("count a flag of a continent striking back together under the continent's", () => {
        const european = (flag: string) => flag === "fr" || flag === "de" ? "eu" : flag

        expect(namedSidesOf(cast, [5], cast.takes, ["eu"], european)).toEqual([{flag: "eu", fighters: [{number: 5, name: "Toto", took: 2}]}])
    })

    it("are three a side at most, and leave out one who took nothing for a side in what the clip plays", () => {
        const named = namedSidesOf(cast, [1, 3, 4, 5, 2], cast.takes, ["fr"], sameFlag)

        expect(named).toHaveLength(1)
        expect(named[0].fighters.map(({number}) => number)).toEqual([1, 3, 4])
        expect(namedSidesOf(cast, [2], cast.takes, ["fr"], sameFlag)).toEqual([])
    })

    it("are nobody when the editor named nobody", () => {
        expect(namedSidesOf(cast, [], cast.takes, ["fr", "tr"], sameFlag)).toEqual([])
    })

    it("refuse a number the cast does not have", () => {
        expect(() => namedSidesOf(cast, [1, 9], cast.takes, ["fr"], sameFlag)).toThrow("no fighter number 9 in the cast")
    })
})
