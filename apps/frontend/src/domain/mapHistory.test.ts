import {describe, expect, it} from "vitest"
import {MapFrame, MapHistory, MapReel} from "./mapHistory.ts"

const frame = (at: number, owners: Record<number, string>): MapFrame => ({
    at,
    tiles: Uint32Array.from(Object.keys(owners).map(Number)),
    owners: Object.values(owners),
})

const history = (frames: MapFrame[]): MapHistory => ({frames, hits: new Map()})

const ownersOf = (reel: MapReel, at: number) =>
    Object.fromEntries([...reel.ownersAt(at)].flatMap((owner, tile) => owner === 0 ? [] : [[tile, reel.codes[owner]]]))

describe("MapReel", () => {
    it("is the opening map, then each frame's changes on top of it", () => {
        const reel = new MapReel(history([frame(0, {1: "fr", 2: "de"}), frame(10, {2: "fr", 3: "jp"}), frame(20, {1: ""})]), 4)

        expect(ownersOf(reel, 0)).toEqual({1: "fr", 2: "de"})
        expect(ownersOf(reel, 1)).toEqual({1: "fr", 2: "fr", 3: "jp"})
        expect(ownersOf(reel, 2)).toEqual({2: "fr", 3: "jp"})
        expect(reel.atOf(1)).toBe(10)
    })

    it("seeks back and forth across its checkpoints to the same maps", () => {
        const frames = [frame(0, {1: "fr"}), ...Array.from({length: 100}, (_, i) => frame(i + 1, {1: i % 2 === 0 ? "de" : "fr", [i + 2]: "jp"}))]
        const reel = new MapReel(history(frames), 200)

        const forward = ownersOf(reel, 70)
        ownersOf(reel, 99)
        expect(ownersOf(reel, 70)).toEqual(forward)
        expect(forward[1]).toBe("fr")
        expect(Object.keys(forward)).toHaveLength(71)
    })

    it("counts each flag's tiles at a frame, most first", () => {
        const reel = new MapReel(history([frame(0, {1: "fr", 2: "de", 3: "de"})]), 3)

        expect(reel.holdingsAt(0)).toEqual([{code: "de", tiles: 2}, {code: "fr", tiles: 1}])
    })
})
