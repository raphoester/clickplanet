import {describe, expect, it} from "vitest"
import {layingOf} from "./mines.ts"

describe("layingOf", () => {
    it("lays a mine on a tile the flag holds", () => {
        expect(layingOf("fr", "fr", false, false)).toBe("lay")
    })

    it("clicks any tile the flag does not hold, a capital too", () => {
        expect(layingOf("jp", "fr", false, false)).toBe("click")
        expect(layingOf(undefined, "fr", false, false)).toBe("click")
        expect(layingOf("jp", "fr", false, true)).toBe("click")
    })

    it("refuses a tile of the flag mined already, or a capital the knock-out plays on", () => {
        expect(layingOf("fr", "fr", true, false)).toBe("refused")
        expect(layingOf("fr", "fr", false, true)).toBe("refused")
    })
})
