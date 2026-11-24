import {describe, expect, it} from "vitest"
import {distanceKm} from "../flags/pin.ts"
import {boardOf} from "./boards.ts"
import {capitalsOf} from "./capitals.ts"
import {EARTH_1, placesOn} from "./testBoards.ts"

describe("capitalsOf", () => {
    it("puts each country's capital at its tile on the board", () => {
        const capitals = capitalsOf(placesOn(EARTH_1), boardOf("earth-1").capitals)
        expect(capitals).toHaveLength(239)
        const at = (country: string) => capitals.find(capital => capital.country === country)!
        // A tile is about 26 km from the next on earth-1: the capital's tile is the one nearest the city.
        expect(distanceKm(at("fr"), {latitude: 48.8566, longitude: 2.3522})).toBeLessThan(26)
        expect(distanceKm(at("ar"), {latitude: -34.6037, longitude: -58.3816})).toBeLessThan(26)
    })

    it("refuses a file of another board, and a tile the board does not hold", () => {
        const places = placesOn(EARTH_1)
        expect(() => capitalsOf(places, JSON.stringify({tiles: 12, capitals: {fr: 1}})))
            .toThrow("the capitals are of a board of 12 tiles, earth-1 has 260503")
        expect(() => capitalsOf(places, JSON.stringify({tiles: 260503, capitals: {fr: 260504}})))
            .toThrow("fr's capital is tile 260504, not one of earth-1")
    })
})
