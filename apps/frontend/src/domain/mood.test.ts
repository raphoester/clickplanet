import {describe, expect, it} from "vitest"
import {ClickBudget} from "../backends/clickBudget.ts"
import {PLAIN_TURN, Turn} from "../backends/turn.ts"
import {moodOf} from "./mood.ts"

const battle: Turn = {...PLAIN_TURN, kind: "battle", entryId: "e", since: 0, endsAt: 1, roundId: "e", times: 3}
const halloween: Turn = {...PLAIN_TURN, kind: "halloween", entryId: "h", since: 0, endsAt: 1}
const budget: ClickBudget = {tokens: 5, capacity: 10, perSecond: 1, readAt: 0}

describe("moodOf", () => {
    it("is the battle's while one plays", () => {
        expect(moodOf(battle, budget)).toBe("battle")
    })

    it("is Halloween's while a Halloween entry plays", () => {
        expect(moodOf(halloween, budget)).toBe("halloween")
    })

    it("is frozen while the rules freeze the map, even during a battle or Halloween", () => {
        expect(moodOf(PLAIN_TURN, {...budget, frozen: true})).toBe("frozen")
        expect(moodOf(battle, {...budget, frozen: true})).toBe("frozen")
        expect(moodOf(halloween, {...budget, frozen: true})).toBe("frozen")
    })

    it("is none otherwise", () => {
        expect(moodOf(PLAIN_TURN, budget)).toBeUndefined()
        expect(moodOf({...battle, endsAt: undefined}, budget)).toBeUndefined()
        expect(moodOf(undefined, undefined)).toBeUndefined()
    })
})
