import {describe, expect, it} from "vitest"
import {wordsOf} from "./wordsOf.ts"

const dancing = {url: "https://static.klipy.com/ii/a/sm.webp", width: 220, height: 124}

describe("a GIF", () => {
    it("is said as GIF where only words fit", () => {
        expect(wordsOf({text: "", gif: dancing})).toBe("GIF")
        expect(wordsOf({text: "message a"})).toBe("message a")
    })
})
