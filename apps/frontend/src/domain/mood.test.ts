import {describe, expect, it} from "vitest"
import {PLAIN_TURN, Turn} from "../backends/turn.ts"
import {moodOf} from "./mood.ts"

const battle: Turn = {...PLAIN_TURN, kind: "battle", entryId: "e", since: 0, endsAt: 1, roundId: "e", times: 3}

describe("moodOf", () => {
    it("is the battle's while one plays", () => {
        expect(moodOf(battle, false)).toBe("battle")
    })

    it("is frozen while the rules freeze the map, even during a battle", () => {
        expect(moodOf(PLAIN_TURN, true)).toBe("frozen")
        expect(moodOf(battle, true)).toBe("frozen")
    })

    it("is none otherwise", () => {
        expect(moodOf(PLAIN_TURN, false)).toBeUndefined()
        expect(moodOf({...battle, endsAt: undefined}, false)).toBeUndefined()
        expect(moodOf(undefined, false)).toBeUndefined()
    })
})
