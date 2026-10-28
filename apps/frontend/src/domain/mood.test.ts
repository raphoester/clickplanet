import {describe, expect, it} from "vitest"
import {ClickBudget} from "../backends/clickBudget.ts"
import {Battle} from "./battle.ts"
import {moodOf} from "./mood.ts"

const battle: Battle = {entryId: "e", since: 0, endsAt: 1, description: "", roundId: "e"}
const budget: ClickBudget = {tokens: 5, capacity: 10, perSecond: 1, readAt: 0}

describe("moodOf", () => {
    it("is the battle's while one plays", () => {
        expect(moodOf(battle, budget)).toBe("battle")
    })

    it("is frozen while the rules freeze the map, even during a battle", () => {
        expect(moodOf(undefined, {...budget, frozen: true})).toBe("frozen")
        expect(moodOf(battle, {...budget, frozen: true})).toBe("frozen")
    })

    it("is none otherwise", () => {
        expect(moodOf(undefined, budget)).toBeUndefined()
        expect(moodOf(undefined, undefined)).toBeUndefined()
    })
})
