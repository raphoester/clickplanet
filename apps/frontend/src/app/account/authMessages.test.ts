import {describe, expect, it} from "vitest"
import {messageOf, retryOf} from "./authMessages.ts"

describe("messageOf", () => {
    // Sign-in spends the same budget as the click token's mint.
    it("tells the player to wait when the budget is spent", () => {
        expect(messageOf("tooManyTries")).toMatch(/wait/i)
    })

    it("names the service that refused", () => {
        expect(messageOf("refused", "discord")).toMatch(/^Discord /)
        expect(messageOf("refused")).toMatch(/^The service /)
    })
})

describe("retryOf", () => {
    it("sends the same code again only when the server did not use it", () => {
        expect(retryOf("tooManyTries")).toBe("complete")
        expect(retryOf("failed")).toBe("complete")
    })

    // CompleteSignIn clears the flow cookie on these, so the code cannot be used twice.
    it("goes back to the provider when the code is spent", () => {
        expect(retryOf("startAgain")).toBe("start")
        expect(retryOf("refused")).toBe("start")
    })

    it("offers nothing when sign-in cannot work", () => {
        expect(retryOf("off")).toBe("none")
        expect(retryOf("notOffered")).toBe("none")
    })
})
