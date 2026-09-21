// The oracle's rules, against hand-written polygons. No network: what is pinned here is the
// behaviour the map generator depends on, not Natural Earth's coastline.
import {describe, expect, it} from "vitest"

import {SEA, groundIndex} from "./ground.mjs"

const polygon = (rings) => ({
    geometry: {type: "Polygon", coordinates: rings},
    properties: {},
})

const named = (code, rings) => ({
    geometry: {type: "Polygon", coordinates: rings},
    properties: {ISO_A2_EH: code},
})

// A square from (x0,y0) to (x1,y1), counter-clockwise.
const square = (x0, y0, x1, y1) => [[x0, y0], [x1, y0], [x1, y1], [x0, y1], [x0, y0]]

const noIce = {features: []}
const noCountries = {features: []}

describe("groundOf", () => {
    it("answers the country a point falls in, lowercased", () => {
        const ground = groundIndex({
            countries: {features: [named("FR", [square(0, 40, 10, 50)])]},
            iceShelves: noIce,
        })
        expect(ground.groundOf(5, 45)).toBe("fr")
    })

    it("answers SEA outside every polygon", () => {
        const ground = groundIndex({
            countries: {features: [named("FR", [square(0, 40, 10, 50)])]},
            iceShelves: noIce,
        })
        expect(ground.groundOf(20, 45)).toBe(SEA)
    })

    it("answers SEA inside a hole", () => {
        const ground = groundIndex({
            countries: {features: [named("ZA", [square(0, 0, 10, 10), square(4, 4, 6, 6)])]},
            iceShelves: noIce,
        })
        expect(ground.groundOf(1, 1)).toBe("za")
        expect(ground.groundOf(5, 5)).toBe(SEA)
    })

    it("answers the enclave that fills another country's hole", () => {
        const ground = groundIndex({
            countries: {
                features: [
                    named("ZA", [square(0, 0, 10, 10), square(4, 4, 6, 6)]),
                    named("LS", [square(4, 4, 6, 6)]),
                ],
            },
            iceShelves: noIce,
        })
        expect(ground.groundOf(5, 5)).toBe("ls")
    })

    it("takes a multipolygon's pieces as separate shapes", () => {
        const ground = groundIndex({
            countries: {
                features: [{
                    geometry: {
                        type: "MultiPolygon",
                        coordinates: [[square(0, 0, 2, 2)], [square(20, 20, 22, 22)]],
                    },
                    properties: {ISO_A2_EH: "FR"},
                }],
            },
            iceShelves: noIce,
        })
        expect(ground.groundOf(1, 1)).toBe("fr")
        expect(ground.groundOf(21, 21)).toBe("fr")
        expect(ground.groundOf(11, 11)).toBe(SEA)
    })
})

describe("the countries Natural Earth does not code", () => {
    const uncoded = (adm0A3, rings) => ({
        geometry: {type: "Polygon", coordinates: rings},
        properties: {ISO_A2_EH: "-99", ADM0_A3: adm0A3},
    })

    it("folds Somaliland into Somalia and Northern Cyprus into Cyprus", () => {
        const ground = groundIndex({
            countries: {features: [uncoded("SOL", [square(0, 0, 2, 2)]), uncoded("CYN", [square(10, 10, 12, 12)])]},
            iceShelves: noIce,
        })
        expect(ground.groundOf(1, 1)).toBe("so")
        expect(ground.groundOf(11, 11)).toBe("cy")
    })

    // India and Pakistan both claim it and the dataset declines to pick, so there is no country to
    // fold it into. Four lattice vertices, and they stay sea rather than being handed to a side.
    it("leaves a territory with no ADM0_ISO as sea", () => {
        const ground = groundIndex({
            countries: {features: [uncoded("KAS", [square(0, 0, 2, 2)])]},
            iceShelves: noIce,
        })
        expect(ground.groundOf(1, 1)).toBe(SEA)
    })
})

describe("the antarctic ice shelves", () => {
    it("are ground, and are Antarctica", () => {
        const ground = groundIndex({
            countries: noCountries,
            iceShelves: {features: [polygon([square(-10, -80, 10, -75)])]},
        })
        expect(ground.groundOf(0, -78)).toBe("aq")
    })

    // A shelf runs onto a claimed coast in places. The country is asked first so the flag there is
    // the country's, not bare Antarctica's.
    it("lose to a country where the two overlap", () => {
        const ground = groundIndex({
            countries: {features: [named("CL", [square(-10, -80, 0, -75)])]},
            iceShelves: {features: [polygon([square(-10, -80, 10, -75)])]},
        })
        expect(ground.groundOf(-5, -78)).toBe("cl")
        expect(ground.groundOf(5, -78)).toBe("aq")
    })
})

// Natural Earth cuts Antarctica down the antimeridian, so a point exactly on 180 lands on the
// polygon's own edge where an even-odd test has no answer. Nudging off the cut is what keeps the
// row of tiles straight out from the south pole inside a country.
describe("the antimeridian seam", () => {
    const ground = groundIndex({
        countries: {features: [named("AQ", [square(-180, -90, 180, -60)])]},
        iceShelves: noIce,
    })

    it("answers on the cut itself, from either side", () => {
        expect(ground.groundOf(180, -70)).toBe("aq")
        expect(ground.groundOf(-180, -70)).toBe("aq")
    })

    it("still answers sea beyond the polygon", () => {
        expect(ground.groundOf(180, -50)).toBe(SEA)
    })
})
