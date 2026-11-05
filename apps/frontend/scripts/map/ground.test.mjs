import {describe, expect, it} from "vitest"

import {SEA, groundIndex, mostlyLake} from "./ground.mjs"

const polygon = (rings) => ({
    geometry: {type: "Polygon", coordinates: rings},
    properties: {},
})

const named = (code, rings) => ({
    geometry: {type: "Polygon", coordinates: rings},
    properties: {ISO_A2_EH: code},
})

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

    it("lose to a country where the two overlap", () => {
        const ground = groundIndex({
            countries: {features: [named("CL", [square(-10, -80, 0, -75)])]},
            iceShelves: {features: [polygon([square(-10, -80, 10, -75)])]},
        })
        expect(ground.groundOf(-5, -78)).toBe("cl")
        expect(ground.groundOf(5, -78)).toBe("aq")
    })
})

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

describe("the lakes", () => {
    const lake = (name, rings) => ({geometry: {type: "Polygon", coordinates: rings}, properties: {name}})
    // A spacing of one degree, so a cell reaches about half a degree from its vertex.
    const SPACING = Math.PI / 180
    const ground = groundIndex({
        countries: {features: [named("US", [square(-100, 30, -60, 60)])]},
        iceShelves: noIce,
        lakes: {features: [
            lake("Lake Superior", [square(-90, 46, -85, 49), square(-89, 47, -88, 48)]),
            lake("A reservoir down a valley", [square(-70, 40, -69.8, 50)]),
            lake("Lake Eyre North", [square(-80, 35, -75, 38)]),
        ]},
    })

    it("leave the country's ground to the country: the oracle does not cut them out of it", () => {
        expect(ground.groundOf(-87, 46.5)).toBe("us")
        expect(ground.lakeAt(-87, 46.5)).toBe(true)
    })

    it("make a cell water when most of it is lake", () => {
        expect(mostlyLake(ground, -86, 46.5, SPACING)).toBe(true)
    })

    it("leave a cell land when the lake covers less than half of it, at its shore or across a narrow lake", () => {
        expect(mostlyLake(ground, -85, 47.5, SPACING)).toBe(false)
        expect(mostlyLake(ground, -69.9, 45, SPACING)).toBe(false)
        expect(ground.lakeAt(-69.9, 45)).toBe(true)
    })

    it("keep an island in a lake as land", () => {
        expect(mostlyLake(ground, -88.5, 47.5, SPACING)).toBe(false)
    })

    it("are not water where they are usually dry", () => {
        expect(ground.lakeAt(-77, 36)).toBe(false)
        expect(mostlyLake(ground, -77, 36, SPACING)).toBe(false)
    })
})
