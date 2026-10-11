import {describe, expect, it} from "vitest"
import {bugContextOf, BUILD, buildOf} from "./bugContext.ts"

describe("buildOf", () => {
    it("is the file a module runs from, which carries the build's hash", () => {
        expect(buildOf("https://clickplanet.lol/assets/play-a1B2c3D4.js")).toBe("play-a1B2c3D4.js")
        expect(buildOf("http://localhost:5173/src/app/support/bugContext.ts?t=1")).toBe("bugContext.ts")
    })

    it("is empty for what is not a URL", () => {
        expect(buildOf("not a url")).toBe("")
    })
})

describe("bugContextOf", () => {
    it("says the build, the size of the page in whole pixels and whether the server was lost", () => {
        expect(bugContextOf({innerWidth: 390.4, innerHeight: 843.6}, true)).toEqual({
            build: BUILD,
            viewportWidth: 390,
            viewportHeight: 844,
            connectionDown: true,
        })
        expect(BUILD).toBe("bugContext.ts")
    })
})
