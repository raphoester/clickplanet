import {describe, expect, it} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {failureOf} from "./refusal.ts"

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
