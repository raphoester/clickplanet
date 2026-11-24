import {describe, expect, it} from "vitest"
import {shopReturnOf, withoutShopReturn} from "./shopReturn.ts"

const ORIGIN = "https://clickplanet.lol"

describe("the way back from Stripe's page", () => {
    it("reads a payment, or a page left", () => {
        expect(shopReturnOf(new URL("/play?shop=paid", ORIGIN))).toBe("paid")
        expect(shopReturnOf(new URL("/play?shop=cancelled", ORIGIN))).toBe("cancelled")
    })

    it("reads nothing of a plain page, or of another value", () => {
        expect(shopReturnOf(new URL("/play", ORIGIN))).toBeUndefined()
        expect(shopReturnOf(new URL("/play?shop=free", ORIGIN))).toBeUndefined()
    })

    it("leaves the address bar, and keeps the rest of the query and the hash", () => {
        const stripped = withoutShopReturn(new URL("/play?f=de&shop=paid&s=t#home", ORIGIN))

        expect(stripped.search).toBe("?f=de&s=t")
        expect(stripped.hash).toBe("#home")
        expect(shopReturnOf(stripped)).toBeUndefined()
    })
})
