import {describe, expect, it} from "vitest"
import {orderText} from "./orderParts.tsx"

describe("orderText", () => {
    it("says how many tiles to take and the countries the order names", () => {
        expect(orderText({kind: "takeFor", country: "fr", goal: 60, taken: 0})).toBe("Take 60 tiles for France")
        expect(orderText({kind: "takeFor", country: "fr", goal: 1, taken: 0})).toBe("Take a tile for France")
        expect(orderText({kind: "takeFrom", country: "dz", goal: 40, taken: 0})).toBe("Take 40 tiles from Algeria")
        expect(orderText({kind: "takeIn", country: "br", goal: 200, taken: 0})).toBe("Take 200 tiles in Brazil")
        expect(orderText({kind: "takeFromIn", country: "dz", ground: "br", goal: 60, taken: 0})).toBe("Take 60 tiles from Algeria in Brazil")
        expect(orderText({kind: "breakShields", country: "pw", goal: 1450, reached: 0})).toBe("Break 1,450 shields of Palau")
    })

    it("says what the other kinds ask, one or many", () => {
        expect(orderText({kind: "catchBoxes", goal: 4, reached: 0})).toBe("Catch 4 boxes")
        expect(orderText({kind: "answerQuizzes", goal: 1, reached: 0})).toBe("Answer a quiz right")
        expect(orderText({kind: "bombIn", country: "br", goal: 1, reached: 0})).toBe("Drop a bomb on Brazil")
        expect(orderText({kind: "bombIn", country: "br", goal: 2, reached: 0})).toBe("Drop 2 bombs on Brazil")
        expect(orderText({kind: "shieldTiles", shields: 10, goal: 2, reached: 0})).toBe("Put 10 shields on 2 tiles")
        expect(orderText({kind: "shieldTiles", shields: 10, goal: 1, reached: 0})).toBe("Put 10 shields on a tile")
        expect(orderText({kind: "fortifyTiles", goal: 40, reached: 0})).toBe("Fortify 40 tiles")
    })
})
