import {describe, expect, it} from "vitest"
import {firstColumn, wordsOf} from "./sheet.mjs"

describe("firstColumn", () => {
    it("reads the first cell of every row", () => {
        expect(firstColumn("a,,\nb,x,y\r\nc,,\n")).toEqual(["a", "b", "c"])
    })

    it("reads a quoted cell with commas, quotes and line breaks", () => {
        expect(firstColumn("\"a,b\",,\n\"say \"\"hi\"\"\",,\n#(),,\"note\non two lines\"\nz,,")).toEqual([
            "a,b",
            "say \"hi\"",
            "#()",
            "z",
        ])
    })

    it("reads a last row with no line break", () => {
        expect(firstColumn("a\nb")).toEqual(["a", "b"])
    })
})

describe("wordsOf", () => {
    it("drops blanks and repeats and makes the spaces single", () => {
        expect(wordsOf(["ass", "", "  ", "pain  in\nthe ass ", "ass"])).toEqual(["ass", "pain in the ass"])
    })
})
