import {describe, expect, it} from "vitest"
import {EXPERIMENTS, variantsOf} from "./experiments.ts"

describe("the arms a clip is made in", () => {
    it("are none when the job gave none", () => {
        expect(variantsOf(null)).toEqual({})
        expect(variantsOf("")).toEqual({})
        expect(variantsOf("{}")).toEqual({})
    })

    it("are the arm of each experiment, as the job wrote them", () => {
        expect(variantsOf(`{"sound": "trending"}`)).toEqual({sound: "trending"})
        expect(variantsOf(`{"sound": "anthem"}`)).toEqual({sound: "anthem"})
    })

    it("refuse an experiment the generator does not know, so no clip is counted in an arm it was not made in", () => {
        expect(() => variantsOf(`{"end-card": "podium"}`)).toThrow(`no experiment "end-card"`)
        expect(() => variantsOf(`{"length": "short"}`)).toThrow(`no experiment "length"`)
        expect(() => variantsOf(`{"constructor": "short"}`)).toThrow(`no experiment "constructor"`)
    })

    it("refuse an arm the experiment does not have", () => {
        expect(() => variantsOf(`{"sound": "loud"}`)).toThrow(`no arm "loud" in the experiment "sound": anthem, trending`)
        expect(() => variantsOf(`{"sound": 8.5}`)).toThrow(`no arm 8.5`)
        expect(() => variantsOf(`{"sound": "fell-back"}`)).toThrow(`no arm "fell-back" in the experiment "sound": anthem, trending`)
    })

    it("refuse what is not an object of arms", () => {
        expect(() => variantsOf(`["short"]`)).toThrow("not an object")
        expect(() => variantsOf(`null`)).toThrow("not an object")
        expect(() => variantsOf(`{length`)).toThrow()
    })

    it("name every arm in words", () => {
        for (const arms of Object.values(EXPERIMENTS)) {
            for (const arm of arms) expect(arm).toMatch(/^[a-z]+(-[a-z]+)*$/)
        }
    })
})
