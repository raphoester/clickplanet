import {describe, expect, it} from "vitest"
import {callbackOf} from "./callback.ts"

const at = (path: string) => callbackOf(new URL(path, "https://backoffice.clickplanet.lol"))

describe("callbackOf", () => {
    it("reads the code and the state Google sent back", () => {
        expect(at("/auth/callback?code=abc&state=xyz")).toEqual({kind: "code", code: "abc", state: "xyz"})
        expect(at("/auth/callback/?state=xyz&code=abc")).toEqual({kind: "code", code: "abc", state: "xyz"})
    })

    it("reads a refusal, which wins over a code", () => {
        expect(at("/auth/callback?error=access_denied")).toEqual({kind: "refused"})
        expect(at("/auth/callback?error=access_denied&code=abc&state=xyz")).toEqual({kind: "refused"})
    })

    it("reads a link with a part missing as broken", () => {
        expect(at("/auth/callback")).toEqual({kind: "broken"})
        expect(at("/auth/callback?code=abc")).toEqual({kind: "broken"})
        expect(at("/auth/callback?state=xyz")).toEqual({kind: "broken"})
    })

    it("is no callback anywhere else", () => {
        expect(at("/")).toBeUndefined()
        expect(at("/?code=abc&state=xyz")).toBeUndefined()
        expect(at("/auth/callbacks?code=abc&state=xyz")).toBeUndefined()
    })
})
