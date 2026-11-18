import {describe, expect, it} from "vitest"
import {Era, eraAt, ErasReel, FIRST_ERA, lastEraOf, piecesOf} from "./eras.ts"
import {MapFrame, MapHistory} from "./mapHistory.ts"
import {EARTH_1, FIXTURE_1} from "../testdata/boards.ts"

const frame = (at: number, owners: Record<number, string>): MapFrame => ({
    at,
    tiles: Uint32Array.from(Object.keys(owners).map(Number)),
    owners: Object.values(owners),
    changedAt: new Float64Array(Object.keys(owners).length).fill(at),
})

const tilesOf = (board: string) => board === FIXTURE_1.name ? FIXTURE_1.tiles : EARTH_1.tiles

const ownersOf = (owners: Uint16Array, codes: readonly string[]) =>
    Object.fromEntries([...owners].flatMap((owner, tile) => owner === 0 ? [] : [[tile, codes[owner]]]))

// earth-1 until 20, then fixture-1: its first frame, at the cut, holds every tile held on it, under ids earth-1 used too.
const ERAS: Era[] = [{startedAt: 0, board: EARTH_1.name}, {startedAt: 20, board: FIXTURE_1.name}]
const ACROSS: MapHistory = {
    frames: [
        frame(0, {1: "fr", 2: "de", 3000: "jp"}),
        frame(10, {2: "fr"}),
        frame(20, {1: "it", 5: "es"}),
        frame(30, {5: ""}),
    ],
    hits: [new Map([[2, 3]]), new Map([[1, 4]])],
    eras: ERAS,
}

describe("eraAt", () => {
    it("is the last era to start at the time or before, and the first before them all", () => {
        expect(eraAt(ERAS, -5)).toBe(0)
        expect(eraAt(ERAS, 0)).toBe(0)
        expect(eraAt(ERAS, 19)).toBe(0)
        expect(eraAt(ERAS, 20)).toBe(1)
        expect(eraAt(ERAS, 1e12)).toBe(1)
    })
})

describe("piecesOf", () => {
    it("draws each frame on the board of the era at its time, the frames before an era's start on the board before", () => {
        const pieces = piecesOf(ACROSS)

        expect(pieces.map(({era, board, from, to, startedAt}) => ({era, board, from, to, startedAt}))).toEqual([
            {era: 0, board: "earth-1", from: 0, to: 2, startedAt: 0},
            {era: 1, board: "fixture-1", from: 2, to: 4, startedAt: 20},
        ])
        expect(pieces[1].history.frames.map(({at}) => at)).toEqual([20, 30])
        expect(pieces[1].history.eras).toEqual([ERAS[1]])
    })

    it("hands each era its own hits, by the tiles of its own board", () => {
        const [earth, fixture] = piecesOf(ACROSS)

        expect(earth.history.hits).toEqual([new Map([[2, 3]])])
        expect(fixture.history.hits).toEqual([new Map([[1, 4]])])
    })

    it("leaves out an era with no frame in the window", () => {
        const eras: Era[] = [{startedAt: 0, board: "earth-1"}, {startedAt: 12, board: "siege-1"}, {startedAt: 18, board: "earth-1"}]
        const pieces = piecesOf({frames: [frame(0, {1: "fr"}), frame(10, {}), frame(20, {1: "de"})], hits: [], eras})

        expect(pieces.map(({era, board}) => [era, board])).toEqual([[0, "earth-1"], [2, "earth-1"]])
    })

    it("is the history as it is with one era", () => {
        const one: MapHistory = {frames: ACROSS.frames.slice(0, 2), hits: [new Map([[2, 3]])], eras: [FIRST_ERA]}
        const [only, ...none] = piecesOf(one)

        expect(none).toEqual([])
        expect(only.history).toEqual(one)
        expect(lastEraOf(one)).toEqual({...one, board: "earth-1"})
    })
})

describe("ErasReel", () => {
    const reel = new ErasReel(ACROSS, tilesOf)

    it("names the board each frame is drawn on, and its place on that board's reel", () => {
        expect([0, 1, 2, 3].map((at) => {
            const {piece, frame} = reel.pieceAt(at)
            return [piece.board, frame]
        })).toEqual([["earth-1", 0], ["earth-1", 1], ["fixture-1", 0], ["fixture-1", 1]])
        expect(reel.length).toBe(4)
        expect(reel.atOf(2)).toBe(20)
    })

    it("clears the map at the cut and draws the next board whole: nothing of the board before is carried", () => {
        const before = reel.pieceAt(1)
        expect(ownersOf(before.piece.reel.ownersAt(before.frame), before.piece.reel.codes)).toEqual({1: "fr", 2: "fr", 3000: "jp"})

        const cut = reel.pieceAt(2)
        expect(ownersOf(cut.piece.reel.ownersAt(cut.frame), cut.piece.reel.codes)).toEqual({1: "it", 5: "es"})
        const after = reel.pieceAt(3)
        expect(ownersOf(after.piece.reel.ownersAt(after.frame), after.piece.reel.codes)).toEqual({1: "it"})
    })

    it("sizes each era's reel on its own board", () => {
        expect(reel.pieces.map(({reel: own}) => own.ownersAt(0).length)).toEqual([EARTH_1.tiles + 1, FIXTURE_1.tiles + 1])
    })

    it("cuts where the next era's first frame is, at the era's start", () => {
        expect(reel.cuts()).toEqual([{frame: 2, at: 20, board: "fixture-1"}])
        expect(new ErasReel({...ACROSS, frames: ACROSS.frames.slice(0, 2)}, tilesOf).cuts()).toEqual([])
    })

    it("captions the cut from its frame for as many frames as asked, and nowhere else", () => {
        expect([0, 1, 2, 3].map((frame) => reel.passingCut(frame, 2))).toEqual([false, false, true, true])
        expect(reel.passingCut(3, 1)).toBe(false)
    })

    it("plays the map the window ends on as its last era alone", () => {
        const last = lastEraOf(ACROSS)

        expect(last.board).toBe("fixture-1")
        expect(last.frames.map(({at}) => at)).toEqual([20, 30])
        expect(last.hits).toEqual([new Map([[1, 4]])])
    })
})
