import {describe, expect, it} from "vitest"
import {HOLD_MS, holdOf, NO_POPS, poppedIn, poppedOut, TILES_HOLD_MS} from "./xpPops.ts"

describe("poppedIn", () => {
    it("shows the first gain at once", () => {
        expect(poppedIn(NO_POPS, [{source: "boxesCaught", xp: 30}])).toEqual({
            shown: {source: "boxesCaught", xp: 30, key: 1, bumps: 0}, waiting: [], next: 2,
        })
    })

    it("counts the tiles up while the player clicks, and bumps at each take", () => {
        let pops = poppedIn(NO_POPS, [{source: "tilesTaken", xp: 1}])
        pops = poppedIn(pops, [{source: "tilesTaken", xp: 1}])
        pops = poppedIn(pops, [{source: "tilesTaken", xp: 3}])

        expect(pops.shown).toEqual({source: "tilesTaken", xp: 5, key: 1, bumps: 2})
        expect(pops.waiting).toEqual([])
    })

    it("shows the day's first take before its tile", () => {
        const pops = poppedIn(NO_POPS, [{source: "tilesTaken", xp: 1}, {source: "daysPlayed", xp: 100}])

        expect(pops.shown?.source).toBe("daysPlayed")
        expect(pops.waiting.map((pop) => pop.source)).toEqual(["tilesTaken"])
    })

    it("takes the line from the tiles at once, and the tiles keep counting while they wait", () => {
        let pops = poppedIn(NO_POPS, [{source: "tilesTaken", xp: 4}])
        pops = poppedIn(pops, [{source: "bombsDropped", xp: 50}])
        pops = poppedIn(pops, [{source: "tilesTaken", xp: 2}])

        expect(pops.shown).toEqual({source: "bombsDropped", xp: 50, key: 2, bumps: 0})
        expect(pops.waiting).toEqual([{source: "tilesTaken", xp: 6, key: 1, bumps: 1}])
    })

    it("lets another source wait its turn behind one that is not the tiles", () => {
        let pops = poppedIn(NO_POPS, [{source: "missionsDone", xp: 540}])
        pops = poppedIn(pops, [{source: "boxesCaught", xp: 30}])

        expect(pops.shown?.source).toBe("missionsDone")
        expect(pops.waiting.map((pop) => pop.source)).toEqual(["boxesCaught"])
    })

    it("adds a source to its pop, shown or waiting", () => {
        let pops = poppedIn(NO_POPS, [{source: "tilesFortified", xp: 40}])
        pops = poppedIn(pops, [{source: "boxesCaught", xp: 30}])
        pops = poppedIn(pops, [{source: "tilesFortified", xp: 12}, {source: "boxesCaught", xp: 30}])

        expect(pops.shown).toMatchObject({source: "tilesFortified", xp: 52})
        expect(pops.waiting).toMatchObject([{source: "boxesCaught", xp: 60}])
    })

    it("shows nothing for no XP", () => {
        expect(poppedIn(NO_POPS, [{source: "tilesFortified", xp: 0}])).toBe(NO_POPS)
    })
})

describe("poppedOut", () => {
    it("shows the next one waiting, then nothing", () => {
        let pops = poppedIn(NO_POPS, [{source: "tilesTaken", xp: 1}, {source: "daysPlayed", xp: 100}])

        pops = poppedOut(pops)
        expect(pops.shown?.source).toBe("tilesTaken")

        pops = poppedOut(pops)
        expect(pops.shown).toBeUndefined()
        expect(pops.waiting).toEqual([])
    })
})

describe("holdOf", () => {
    it("lets the tiles go sooner than the rest", () => {
        expect(holdOf({source: "tilesTaken", xp: 1, key: 1, bumps: 0})).toBe(TILES_HOLD_MS)
        expect(holdOf({source: "daysPlayed", xp: 100, key: 1, bumps: 0})).toBe(HOLD_MS)
    })
})
