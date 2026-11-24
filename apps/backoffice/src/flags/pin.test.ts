import {describe, expect, it} from "vitest"
import {degreesOf, distanceKm, globeLinkOf, kmWords, pastedPlaceOf, pinProblems} from "./pin.ts"

describe("a pin as typed", () => {
    it("reads a degree with a point or a comma, and nothing else", () => {
        expect(degreesOf(" 48.137 ")).toBe(48.137)
        expect(degreesOf("-1,6778")).toBe(-1.6778)
        expect(degreesOf("")).toBeUndefined()
        expect(degreesOf("north")).toBeUndefined()
    })

    it("splits a latitude and a longitude pasted together, as a map copies them", () => {
        expect(pastedPlaceOf("48.1173, -1.6778")).toEqual({latitude: 48.1173, longitude: -1.6778})
        expect(pastedPlaceOf("48.1173 -1.6778")).toEqual({latitude: 48.1173, longitude: -1.6778})
        expect(pastedPlaceOf("48.1173")).toBeUndefined()
        expect(pastedPlaceOf("91, 0")).toBeUndefined()
    })

    it("refuses a place off the globe", () => {
        expect(pinProblems({latitude: 90, longitude: -180})).toEqual([])
        expect(pinProblems({latitude: -90.1, longitude: 180.1})).toEqual(["A latitude from −90 to 90.", "A longitude from −180 to 180."])
        expect(pinProblems({latitude: NaN, longitude: 0})).toEqual(["A latitude from −90 to 90."])
    })
})

describe("distanceKm", () => {
    it("measures the great circle", () => {
        const paris = {latitude: 48.8566, longitude: 2.3522}
        const london = {latitude: 51.5074, longitude: -0.1278}
        expect(distanceKm(paris, london)).toBeCloseTo(343.6, 0)
        expect(distanceKm(paris, paris)).toBe(0)
        expect(distanceKm({latitude: 0, longitude: 179.5}, {latitude: 0, longitude: -179.5})).toBeCloseTo(111.2, 0)
    })

    it("says a distance in whole km, or to the tenth under 10", () => {
        expect(kmWords(1234.4)).toBe("1,234 km")
        expect(kmWords(4.26)).toBe("4.3 km")
    })
})

describe("globeLinkOf", () => {
    it("turns the game's globe to the place", () => {
        expect(globeLinkOf({latitude: 48.13743, longitude: -1.6})).toBe("https://clickplanet.lol/play?lat=48.1374&lon=-1.6000&zoom=20")
    })
})
