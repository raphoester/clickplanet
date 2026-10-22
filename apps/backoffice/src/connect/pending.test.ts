import {describe, expect, it} from "vitest"
import {requestOf} from "./pending.ts"

const at = (path: string) => requestOf(new URL(path, "https://backoffice.clickplanet.lol"))

describe("requestOf", () => {
    it("reads the request the authorize endpoint sent the browser with", () => {
        expect(at("/connect?request=sealed")).toBe("sealed")
        expect(at("/connect/?request=sealed")).toBe("sealed")
    })

    it("is an empty request on the page with none, which the server then refuses", () => {
        expect(at("/connect")).toBe("")
    })

    it("is no request anywhere else", () => {
        expect(at("/")).toBeUndefined()
        expect(at("/?request=sealed")).toBeUndefined()
        expect(at("/auth/callback?request=sealed")).toBeUndefined()
    })
})
