import {describe, expect, it} from "vitest"

import {SEA} from "./ground.mjs"
import {SEATS, positionOf, seat} from "./seats.mjs"

// A strip of five vertices along the equator, a degree apart, each touching the next.
const strip = (grounds) => {
    const positions = new Float32Array(grounds.length * 3)
    grounds.forEach((_, v) => positions.set(positionOf([v, 0]), v * 3))
    const at = new Uint32Array(grounds.length + 1)
    const to = []
    for (let v = 0; v < grounds.length; v++) {
        if (v > 0) to.push(v - 1)
        if (v < grounds.length - 1) to.push(v + 1)
        at[v + 1] = to.length
    }
    return {grounds: [...grounds], positions, edges: {at, to: Uint32Array.from(to)}}
}

describe("seat", () => {
    it("gives an island the sea vertex nearest its point", () => {
        const {grounds, positions, edges} = strip(["fr", SEA, SEA, SEA, SEA])
        const placed = seat(grounds, positions, edges, {nr: {at: [3.2, 0]}})

        expect(placed).toEqual([{code: "nr", vertex: 3, was: SEA}])
        expect(grounds).toEqual(["fr", SEA, SEA, "nr", SEA])
    })

    it("takes the neighbour's vertex on the coast for a country on the sea", () => {
        const {grounds, positions, edges} = strip(["fr", "fr", "fr", SEA, SEA])
        seat(grounds, positions, edges, {mc: {at: [3.4, 0], takes: "fr", coast: true}})

        expect(grounds).toEqual(["fr", "fr", "mc", SEA, SEA])
    })

    it("takes an inland vertex for a country with no sea, even past a nearer one on the coast", () => {
        const {grounds, positions, edges} = strip(["it", "it", "it", SEA, SEA])
        seat(grounds, positions, edges, {sm: {at: [2.1, 0], takes: "it", coast: false}})

        expect(grounds).toEqual(["it", "sm", "it", SEA, SEA])
    })

    it("refuses an island seat that lands on a country's ground", () => {
        const {grounds, positions, edges} = strip(["fr", SEA, SEA, SEA, SEA])
        expect(() => seat(grounds, positions, edges, {mc: {at: [0.1, 0]}})).toThrow(/lands on fr's ground/)
    })

    it("refuses a seat for a country that has ground of its own", () => {
        const {grounds, positions, edges} = strip(["fr", SEA, SEA, SEA, SEA])
        expect(() => seat(grounds, positions, edges, {fr: {at: [3, 0]}})).toThrow(/ground of its own/)
    })

    it("refuses two seats on one vertex", () => {
        const {grounds, positions, edges} = strip(["fr", SEA, SEA, SEA, SEA])
        expect(() => seat(grounds, positions, edges, {nr: {at: [3, 0]}, tv: {at: [3.1, 0]}})).toThrow(/one vertex/)
    })

    it("needs to know whether a country that takes land is on the coast", () => {
        const {grounds, positions, edges} = strip(["fr", "fr", SEA, SEA, SEA])
        expect(() => seat(grounds, positions, edges, {mc: {at: [1, 0], takes: "fr"}})).toThrow(/coast/)
    })
})

describe("SEATS", () => {
    it("says of every country that takes a neighbour's ground whether it is on the coast", () => {
        for (const {takes, coast} of Object.values(SEATS)) {
            if (takes !== undefined) expect(typeof coast).toBe("boolean")
        }
    })
})
