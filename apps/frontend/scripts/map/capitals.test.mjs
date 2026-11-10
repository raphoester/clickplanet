import {describe, expect, it} from "vitest"

import {capitalsOf} from "./capitals.mjs"
import {positionOf} from "./seats.mjs"

// Each tile's longitude on the equator and its landmass.
const TILES = [
    [0, 1], [1, 1], [2, 1],
    [10, 2],
    [20, 3], [21, 3], [22, 3], [30, 4],
    [40, 5], [41, 5], [42, 5],
    [50, 6], [51, 6],
    [60, 7], [61, 7],
]
const CODES = ["", "bo", "va", "tf", "tf", "so", "ci", "gl"]

const map = () => {
    const frames = new Float32Array(CODES.length * 5)
    frames.set(positionOf([21, 0]), 3 * 5)
    const totals = new Uint32Array(CODES.length)
    for (const [, landmass] of TILES) totals[landmass]++
    return {
        codes: CODES,
        landmass: Uint16Array.from(TILES, ([, landmass]) => landmass),
        positions: Float32Array.from(TILES.flatMap(([lon]) => positionOf([lon, 0]))),
        frames,
        totals,
    }
}

const place = (properties) => ({properties: {ADM0CAP: 0, CAPALT: 0, POP_MAX: 0, LATITUDE: 0, ...properties}})

const PLACES = {
    features: [
        place({NAME: "Sucre", ISO_A2: "BO", FEATURECLA: "Admin-0 capital", ADM0CAP: 1, POP_MAX: 224838, LONGITUDE: 0.1}),
        place({NAME: "La Paz", ISO_A2: "BO", FEATURECLA: "Admin-0 capital", ADM0CAP: 1, POP_MAX: 1590000, LONGITUDE: 2.1}),
        place({NAME: "Vatican City", ISO_A2: "VA", FEATURECLA: "Admin-0 capital", ADM0CAP: 1, LONGITUDE: 50}),
        place({NAME: "Mogadishu", ISO_A2: "SO", ADM0_A3: "SOM", FEATURECLA: "Admin-0 capital", ADM0CAP: 1,
            POP_MAX: 1100000, LONGITUDE: 40.1}),
        place({NAME: "Hargeisa", ISO_A2: "-99", ADM0_A3: "SOL", FEATURECLA: "Admin-0 capital", ADM0CAP: 1,
            POP_MAX: 2000000, LONGITUDE: 42.1}),
        place({NAME: "Yamoussoukro", ISO_A2: "CI", FEATURECLA: "Admin-0 capital", ADM0CAP: 1, CAPALT: 1,
            POP_MAX: 3000000, LONGITUDE: 51.1}),
        place({NAME: "Abidjan", ISO_A2: "CI", FEATURECLA: "Admin-0 capital", ADM0CAP: 1, POP_MAX: 200000,
            LONGITUDE: 49.9}),
        place({NAME: "Nuuk", ISO_A2: "GL", FEATURECLA: "Admin-0 region capital", POP_MAX: 15000, LONGITUDE: 61.1}),
        place({NAME: "Sisimiut", ISO_A2: "GL", FEATURECLA: "Populated place", POP_MAX: 50000, LONGITUDE: 59.9}),
    ],
}

const COUNTRIES = {features: [{properties: {ADM0_A3: "SOL", ISO_A2_EH: "-99", ISO_A2: "-99"}}]}

describe("capitalsOf", () => {
    const capitals = capitalsOf(PLACES, COUNTRIES, map())

    it("gives every country of the map a capital", () => {
        expect(Object.keys(capitals)).toEqual(["bo", "ci", "gl", "so", "tf", "va"])
    })

    it("takes the tile of the country nearest its capital, the bigger city of two main capitals", () => {
        expect(capitals.bo).toBe(3)
    })

    it("takes the main capital over the alternate one, however big", () => {
        expect(capitals.ci).toBe(12)
    })

    it("takes a territory's capital over a bigger town", () => {
        expect(capitals.gl).toBe(15)
    })

    it("takes the country's own capital over the capital of a land it absorbs", () => {
        expect(capitals.so).toBe(9)
    })

    it("starts a country with no place in the middle of its biggest landmass", () => {
        expect(capitals.tf).toBe(6)
    })

    it("starts a country of one tile on it", () => {
        expect(capitals.va).toBe(4)
    })
})
