import {describe, expect, it} from "vitest"
import {callbackOf} from "./signInCallback.ts"

const at = (url: string) => callbackOf(new URL(url, "https://clickplanet.lol"))

describe("callbackOf", () => {
    it("reads the code and the state the provider sent", () => {
        expect(at("/auth/callback?code=abc&state=xyz&scope=openid"))
            .toEqual({kind: "code", code: "abc", state: "xyz"})
    })

    it("keeps a code whose characters were escaped", () => {
        expect(at("/auth/callback?code=4%2F0Ab%3D&state=s"))
            .toEqual({kind: "code", code: "4/0Ab=", state: "s"})
    })

    it("accepts a trailing slash", () => {
        expect(at("/auth/callback/?code=abc&state=xyz")?.kind).toBe("code")
    })

    // OAuth sends error=access_denied when the player cancels on the provider's page.
    it("reads a refusal as declined, even with a code beside it", () => {
        expect(at("/auth/callback?error=access_denied&state=xyz")).toEqual({kind: "declined"})
        expect(at("/auth/callback?error=access_denied&code=abc&state=xyz")).toEqual({kind: "declined"})
    })

    it("is invalid with a part missing", () => {
        expect(at("/auth/callback")).toEqual({kind: "invalid"})
        expect(at("/auth/callback?code=abc")).toEqual({kind: "invalid"})
        expect(at("/auth/callback?state=xyz")).toEqual({kind: "invalid"})
        expect(at("/auth/callback?code=&state=xyz")).toEqual({kind: "invalid"})
    })

    it("is nothing on any other page", () => {
        expect(at("/")).toBeUndefined()
        expect(at("/?code=abc&state=xyz")).toBeUndefined()
        expect(at("/auth/callbacks?code=abc&state=xyz")).toBeUndefined()
    })
})
