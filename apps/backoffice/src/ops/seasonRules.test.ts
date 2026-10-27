import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {complete, copyOf, draftOf} from "./seasonRules.ts"

describe("a season rule set", () => {
    it("is edited as its round of its own and its points' times", () => {
        const [, rush] = new FakeAdmin().seasonRules.sets

        expect(draftOf(rush)).toEqual({name: "rush", ownRound: true, times: "3"})
        expect(copyOf(rush)).toEqual({name: "", ownRound: true, times: "3"})
    })

    it("is complete with a name and whole times from 1 to 10", () => {
        expect(complete({name: "rush", ownRound: true, times: "3"})).toBe(true)
        expect(complete({name: "calm", ownRound: false, times: "1"})).toBe(true)
        expect(complete({name: " ", ownRound: true, times: "3"})).toBe(false)
        expect(complete({name: "rush", ownRound: true, times: "0"})).toBe(false)
        expect(complete({name: "rush", ownRound: true, times: "11"})).toBe(false)
        expect(complete({name: "rush", ownRound: true, times: "1.5"})).toBe(false)
    })
})
