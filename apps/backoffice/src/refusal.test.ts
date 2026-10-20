import {describe, expect, it} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {failureOf, wordsOf} from "./refusal.ts"

describe("failureOf", () => {
    it("reads no session as signed out", () => {
        expect(failureOf(new ConnectError("no session", Code.Unauthenticated)))
            .toEqual({refusal: "signedOut", detail: "no session"})
    })

    it("reads a refused account as not an operator", () => {
        expect(failureOf(new ConnectError("not an operator", Code.PermissionDenied)).refusal).toBe("notOperator")
    })

    it("reads a sign-in to start again", () => {
        expect(failureOf(new ConnectError("start again", Code.FailedPrecondition)).refusal).toBe("startAgain")
    })

    it("reads anything else as a failure", () => {
        expect(failureOf(new ConnectError("boom", Code.Internal)).refusal).toBe("failed")
        expect(failureOf(new TypeError("Failed to fetch")).refusal).toBe("failed")
    })
})

describe("wordsOf", () => {
    it("says what this action means by not_found and failed_precondition", () => {
        const meanings = {notFound: "Gone.", failedPrecondition: "Off."}
        expect(wordsOf(new ConnectError("x", Code.NotFound), meanings)).toBe("Gone.")
        expect(wordsOf(new ConnectError("x", Code.FailedPrecondition), meanings)).toBe("Off.")
    })

    it("says the server's own words for a code the action gives no meaning", () => {
        expect(wordsOf(new ConnectError("no such ban", Code.NotFound))).toBe("The call failed: no such ban")
    })

    it("quotes the server for a request it refused", () => {
        expect(wordsOf(new ConnectError("a mute names an account", Code.InvalidArgument)))
            .toBe("The server refused it: a mute names an account")
    })

    it("says a session that ended, and an account that is not an operator", () => {
        expect(wordsOf(new ConnectError("x", Code.Unauthenticated)))
            .toBe("Your session ended. Reload the page and sign in again.")
        expect(wordsOf(new ConnectError("x", Code.PermissionDenied))).toBe("This account is not an operator.")
    })

    it("says a call that ran out of time may have done part of its work", () => {
        expect(wordsOf(new ConnectError("x", Code.DeadlineExceeded))).toMatch(/may have done part of it/)
    })

    it("says any other failure with its message", () => {
        expect(wordsOf(new TypeError("Failed to fetch"))).toBe("The call failed: Failed to fetch")
    })
})
