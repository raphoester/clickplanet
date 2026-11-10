import {describe, expect, it} from "vitest"
import {MapFrame, MapHistory} from "../mapHistory.ts"
import {framedTilesOf, sagasOf} from "./saga.ts"

// Tiles 1 to 10 are France, in Europe; 11 to 20 Algeria, in Africa.
const groundOf = (tile: number) => tile <= 10 ? "fr" : tile <= 20 ? "dz" : undefined
const regionOf = (country: string) => ({fr: "Europe", dz: "Africa"} as Record<string, string>)[country]

const HOUR = 3_600_000

const frame = (hour: number, ...held: [number, string][]): MapFrame => ({
    at: hour * HOUR, tiles: Uint32Array.from(held, ([tile]) => tile), owners: held.map(([, owner]) => owner),
    changedAt: new Float64Array(held.length).fill(hour * HOUR),
})

const take = (owner: string, ...tiles: number[]): [number, string][] => tiles.map((tile) => [tile, owner])

const historyOf = (...frames: MapFrame[]): MapHistory => ({frames, hits: new Map()})

const europeOf = (history: MapHistory) => sagasOf(history, 20, groundOf, regionOf).find(({continent}) => continent === "Europe")

describe("a place's saga", () => {
    it("tells each flag that held over half of it, when its conquest began, whom it took it from and whom it fell to", () => {
        const europe = europeOf(historyOf(
            frame(0),
            frame(1, ...take("ie", 1, 2, 3, 4, 5, 6)),
            frame(2, ...take("se", 7, 8)),
            frame(4, ...take("pw", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)),
            frame(5),
        ))

        expect(europe?.reigns).toEqual([
            {
                flag: "ie", pushedAt: 0, since: HOUR, until: 4 * HOUR, peak: 0.6, share: 0, tookFrom: undefined, tookFromShare: 0,
                lostTo: "pw",
            },
            {
                flag: "pw", pushedAt: 2 * HOUR, since: 4 * HOUR, until: undefined, peak: 1, share: 1, tookFrom: "ie",
                tookFromShare: 0.6, lostTo: undefined,
            },
        ])
        expect(europe).toMatchObject({kind: "continent", place: {region: "Europe"}, tiles: 10, holder: "pw", share: 1, since: 0, until: 5 * HOUR})
    })

    it("keeps a reign through a dip under half shorter than 3 hours", () => {
        const europe = europeOf(historyOf(
            frame(0, ...take("ie", 1, 2, 3, 4, 5, 6)),
            frame(1, ...take("pw", 1, 2)),
            frame(3, ...take("ie", 1, 2)),
            frame(4),
        ))

        expect(europe?.reigns.map(({flag, since, until}) => ({flag, since, until}))).toEqual([{flag: "ie", since: 0, until: undefined}])
    })

    it("ends a reign at a dip of 3 hours or more", () => {
        const europe = europeOf(historyOf(
            frame(0, ...take("ie", 1, 2, 3, 4, 5, 6)),
            frame(3, ...take("pw", 1, 2)),
            frame(6, ...take("ie", 1, 2)),
        ))

        expect(europe?.reigns.map(({flag, since, until}) => ({flag, since, until}))).toEqual([
            {flag: "ie", since: 0, until: 3 * HOUR},
            {flag: "ie", since: 6 * HOUR, until: undefined},
        ])
    })

    it("leaves out a reign shorter than 3 hours that ended: a raid", () => {
        const europe = europeOf(historyOf(
            frame(0, ...take("ie", 1, 2, 3, 4, 5, 6)),
            frame(4, ...take("pw", 1, 2, 3, 4, 5, 6, 7)),
            frame(5, ...take("ie", 1, 2, 3, 4, 5, 6)),
            frame(6, ...take("pw", 7, 8, 9, 10)),
        ))

        expect(europe?.reigns.map(({flag, since, until}) => ({flag, since, until}))).toEqual([
            {flag: "ie", since: 0, until: 4 * HOUR},
            {flag: "ie", since: 5 * HOUR, until: undefined},
        ])
    })

    it("tells a country that is its whole continent once, as the continent", () => {
        const icy = (tile: number) => tile <= 1000 ? "aq" : undefined
        const sagas = sagasOf(historyOf(frame(0), frame(1, ...Array.from({length: 600}, (_, i): [number, string] => [i + 1, "br"]))),
            1000, icy, () => "Antarctica")

        expect(sagas.filter(({kind}) => kind !== "world").map(({kind, continent}) => ({kind, continent})))
            .toEqual([{kind: "continent", continent: "Antarctica"}])
    })

    it("has no conquest to date for a flag that held a tenth from the start", () => {
        const europe = europeOf(historyOf(frame(0, ...take("ie", 1, 2)), frame(1, ...take("ie", 3, 4, 5, 6))))

        expect(europe?.reigns[0]).toMatchObject({flag: "ie", pushedAt: undefined, since: HOUR, tookFrom: undefined})
    })

    it("leaves out a place no flag ever held over half of", () => {
        const sagas = sagasOf(historyOf(frame(0, ...take("ie", 11, 12, 13, 14, 15)), frame(1)), 20, groundOf, regionOf)

        expect(sagas.map(({continent}) => continent)).toEqual([])
    })
})

describe("the tiles a saga is framed on", () => {
    it("keeps the landmasses of a quarter of the biggest one or more", () => {
        const landmassOf = (tile: number) => tile <= 8 ? 1 : tile <= 10 ? 2 : 3

        expect(framedTilesOf([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11], landmassOf)).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
    })
})
