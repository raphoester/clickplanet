import {describe, expect, it} from "vitest"
import {durationWord, fill} from "./fill.ts"

const PATHS = ["clicks.tempo", "clicks.perSecond", "clicks.burst", "bonuses.boxes.minInterval", "bonuses.boxes.maxInterval",
    "bonuses.bomb.rings", "bonuses.quiz.offerTtl"]
const urf = {
    clicks: {tempo: 3, perSecond: 0.2, burst: 60},
    bonuses: {boxes: {minInterval: "60s", maxInterval: "90s"}, quiz: {offerTtl: "1.500s"}},
}

describe("fill", () => {
    it("writes the values the set plays, as the server does", () => {
        expect(fill("Clicks refill {clicks.tempo}× faster", urf, PATHS)).toEqual({text: "Clicks refill 3× faster", problems: []})
        expect(fill("A click every {clicks.perSecond} s, a bank of {clicks.burst}", urf, PATHS).text)
            .toBe("A click every 0.2 s, a bank of 60")
        expect(fill("A box every {bonuses.boxes.minInterval} to {bonuses.boxes.maxInterval}", urf, PATHS).text)
            .toBe("A box every 1 min to 90 s")
        expect(fill("Answer within {bonuses.quiz.offerTtl}", urf, PATHS).text).toBe("Answer within 1500 ms")
        expect(fill("No placeholder", urf, PATHS)).toEqual({text: "No placeholder", problems: []})
    })

    it("says what does not fill, and leaves it as written", () => {
        expect(fill("{bonuses.bomb.rings} rings", urf, PATHS)).toEqual({
            text: "{bonuses.bomb.rings} rings", problems: ["The set plays no value at {bonuses.bomb.rings}."],
        })
        expect(fill("{frozen}", urf, PATHS).problems).toEqual(["{frozen} names no number or duration of the rules."])
        expect(fill("an { alone", urf, PATHS).problems).toEqual(["A brace opens or closes no placeholder."])
        expect(fill("{clicks.tempo}", undefined, PATHS).problems).toEqual(["The set plays no value at {clicks.tempo}."])
    })
})

describe("durationWord", () => {
    it("counts in the largest whole unit", () => {
        expect(durationWord(0)).toBe("0 s")
        expect(durationWord(3_600_000)).toBe("1 h")
        expect(durationWord(5_400_000)).toBe("90 min")
        expect(durationWord(120_000)).toBe("2 min")
        expect(durationWord(90_000)).toBe("90 s")
        expect(durationWord(500)).toBe("500 ms")
    })
})
