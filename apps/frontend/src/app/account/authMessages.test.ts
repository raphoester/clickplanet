import {describe, expect, it} from "vitest"
import {messageOf, retryOf} from "./authMessages.ts"

describe("messageOf", () => {
    // Sign-in spends the same budget as the click token's mint.
    it("tells the player to wait when the budget is spent", () => {
        expect(messageOf("tooManyTries")).toMatch(/wait/i)
    })

    it("tells how to move an identity another account uses", () => {
        expect(messageOf("linkedElsewhere", "google")).toBe(
            "This Google account is already used by another ClickPlanet account. "
            + "To move it here: sign in with it, delete that account, then link it here.",
        )
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

    // The same identity would be refused again: only the player can change that.
    it("offers nothing after a refused link", () => {
        expect(retryOf("linkedElsewhere")).toBe("none")
        expect(retryOf("alreadyLinked")).toBe("none")
    })
})
