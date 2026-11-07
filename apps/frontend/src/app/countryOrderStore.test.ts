// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"

afterEach(() => {
    window.localStorage.clear()
    vi.resetModules()
})

const loaded = async () => (await import("./countryOrderStore.ts")).useCountryOrder

describe("the country order", () => {
    it("ranks by the points until one is picked", async () => {
        expect((await loaded()).getState().order).toBe("points")
    })

    it("keeps the order picked across a reload", async () => {
        (await loaded()).getState().setOrder("gain")
        vi.resetModules()

        expect((await loaded()).getState().order).toBe("gain")
    })

    it("falls back to the points on a value it does not know", async () => {
        window.localStorage.setItem("clickplanet-country-order", "names")
        expect((await loaded()).getState().order).toBe("points")
    })
})
