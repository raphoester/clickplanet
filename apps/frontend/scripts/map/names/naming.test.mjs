import {describe, expect, it} from "vitest"

import {namesOf} from "./naming.mjs"

const layer = (names, byLandmass, byCountry) => ({names, byLandmass, byCountry})

const evidence = (landmasses, layers, capitals = {}) => ({landmasses, countryLabels: {}, layers, capitals})

describe("namesOf", () => {
    it("gives the biggest landmass the country's name and each other its own", () => {
        const named = namesOf(evidence(
            [{id: 1, code: "fr", tiles: 100}, {id: 2, code: "fr", tiles: 16}],
            {
                subunit50: layer({a: "Corsica"}, {2: {a: 16}}, {fr: {a: 16}}),
                admin1: layer({b: "Haute-Corse"}, {2: {b: 16}}, {fr: {b: 16}}),
            },
        ), {fr: "France"})

        expect(named.get(1)).toEqual({name: "France", main: true})
        expect(named.get(2)).toEqual({name: "Corsica", main: false})
    })

    it("names no landmass after a feature that covers less than half of it", () => {
        const named = namesOf(evidence(
            [{id: 1, code: "fr", tiles: 100}, {id: 2, code: "fr", tiles: 10}],
            {admin1: layer({a: "Somewhere"}, {2: {a: 4}}, {fr: {a: 4}})},
        ), {fr: "France"})

        expect(named.get(2)).toEqual({name: "France 2", main: false})
    })

    it("numbers the landmasses a name is left to in size order", () => {
        const named = namesOf(evidence(
            [{id: 1, code: "us", tiles: 100}, {id: 2, code: "us", tiles: 10}, {id: 3, code: "us", tiles: 5}, {id: 4, code: "us", tiles: 3}],
            {islandGroup: layer({a: "Aleutian Islands"}, {2: {a: 10}, 3: {a: 5}, 4: {a: 3}}, {us: {a: 18}})},
        ), {us: "United States"})

        expect([2, 3, 4].map((id) => named.get(id)?.name)).toEqual(["Aleutian Islands", "Aleutian Islands 2", "Aleutian Islands 3"])
    })

    it("moves the country's name to the capital's landmass when the biggest has a name of its own", () => {
        const named = namesOf(evidence(
            [{id: 1, code: "id", tiles: 800}, {id: 2, code: "id", tiles: 700}],
            {island: layer({a: "New Guinea"}, {1: {a: 800}}, {id: {a: 800}})},
            {id: 2},
        ), {id: "Indonesia"})

        expect(named.get(2)).toEqual({name: "Indonesia", main: true})
        expect(named.get(1)).toEqual({name: "New Guinea", main: false})
    })

    it("never names another piece after the country", () => {
        const named = namesOf(evidence(
            [{id: 1, code: "my", tiles: 300}, {id: 2, code: "my", tiles: 200}],
            {admin1: layer({a: "Malaysia"}, {2: {a: 200}}, {my: {a: 200}})},
        ), {my: "Malaysia"})

        expect(named.get(2)).toEqual({name: "Malaysia 2", main: false})
    })
})
