import {describe, expect, it} from "vitest"
import {welcomeLineOf} from "./welcome.ts"

describe("the welcome line of a recruit", () => {
    it("names the recruiter by the name the clips let a video show", () => {
        expect(welcomeLineOf("acc-1", new Map([["acc-1", "kutza"]]), "Poland")).toBe("kutza called you to fight for Poland")
    })

    it("is none for a recruiter whose name is not shown, and for no recruiter", () => {
        expect(welcomeLineOf("acc-1", new Map(), "Poland")).toBeUndefined()
        expect(welcomeLineOf(undefined, new Map([["acc-1", "kutza"]]), "Poland")).toBeUndefined()
    })
})
