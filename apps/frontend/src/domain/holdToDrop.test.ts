import {describe, expect, it} from "vitest"
import {HoldToDrop} from "./holdToDrop.ts"

const rules = {holdSeconds: 0.7, tolerancePx: 6}

describe("HoldToDrop", () => {
    it("drops on the spot pressed once the press is held to the end", () => {
        const hold = new HoldToDrop<string>(rules)
        hold.begin(1, 100, 100, 10, "paris")

        expect(hold.tick(10.35).progress).toBeCloseTo(0.5)
        expect(hold.tick(10.7).drop).toBe("paris")
    })

    it("drops once, not on every frame after", () => {
        const hold = new HoldToDrop<string>(rules)
        hold.begin(1, 100, 100, 10, "paris")

        hold.tick(10.7)

        expect(hold.tick(10.8).drop).toBeUndefined()
        expect(hold.holding).toBe(false)
    })

    it("drops nothing on a press let go early, which is every click", () => {
        const hold = new HoldToDrop<string>(rules)
        hold.begin(1, 100, 100, 10, "paris")

        hold.end(1)

        expect(hold.tick(11).drop).toBeUndefined()
    })

    it("treats a press that moves as a drag of the globe", () => {
        const hold = new HoldToDrop<string>(rules)
        hold.begin(1, 100, 100, 10, "paris")

        hold.move(1, 103, 104)
        expect(hold.holding).toBe(true)

        hold.move(1, 110, 100)
        expect(hold.tick(11).drop).toBeUndefined()
    })

    it("ignores a second finger", () => {
        const hold = new HoldToDrop<string>(rules)
        hold.begin(1, 100, 100, 10, "paris")

        hold.move(2, 400, 400)
        hold.end(2)

        expect(hold.tick(10.7).drop).toBe("paris")
    })

    it("reports no progress with nothing pressed", () => {
        expect(new HoldToDrop(rules).tick(5)).toEqual({progress: 0})
    })
})
