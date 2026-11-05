import {describe, expect, it} from "vitest"
import {Placeholder} from "../gen/grpc/promo/v1/admin_pb.ts"
import {durationWord, fill, namesBattle, Values} from "./fill.ts"

const RULES = ["clicks.tempo", "clicks.perSecond", "clicks.burst", "bonuses.boxes.minInterval", "bonuses.boxes.maxInterval",
    "bonuses.bomb.rings", "bonuses.quiz.offerTtl"]
const TRIPLE = [
    new Placeholder({path: "battle.times", example: "3"}),
    new Placeholder({path: "battle.points.first", example: "75"}),
    new Placeholder({path: "battle.points.second", example: "54"}),
    new Placeholder({path: "battle.points.third", example: ""}),
]
const urf: Values = {
    played: {
        clicks: {tempo: 3, perSecond: 0.2, burst: 60},
        bonuses: {boxes: {minInterval: "60s", maxInterval: "90s"}, quiz: {offerTtl: "1.500s"}},
    },
    rules: RULES,
    battle: TRIPLE,
}

describe("fill", () => {
    it("writes the values the set plays, as the server does", () => {
        expect(fill("Clicks refill {clicks.tempo}× faster", urf)).toEqual({text: "Clicks refill 3× faster", problems: []})
        expect(fill("A click every {clicks.perSecond} s, a bank of {clicks.burst}", urf).text)
            .toBe("A click every 0.2 s, a bank of 60")
        expect(fill("A box every {bonuses.boxes.minInterval} to {bonuses.boxes.maxInterval}", urf).text)
            .toBe("A box every 1 min to 90 s")
        expect(fill("Answer within {bonuses.quiz.offerTtl}", urf).text).toBe("Answer within 1500 ms")
        expect(fill("No placeholder", urf)).toEqual({text: "No placeholder", problems: []})
    })

    it("says what does not fill, and leaves it as written", () => {
        expect(fill("{bonuses.bomb.rings} rings", urf)).toEqual({
            text: "{bonuses.bomb.rings} rings", problems: ["The set plays no value at {bonuses.bomb.rings}."],
        })
        expect(fill("{frozen}", urf).problems).toEqual(["{frozen} names no number or duration of the rules or the battle."])
        expect(fill("an { alone", urf).problems).toEqual(["A brace opens or closes no placeholder."])
        expect(fill("{clicks.tempo}", {...urf, played: undefined}).problems).toEqual(["The set plays no value at {clicks.tempo}."])
    })

    it("writes the example battle's values as the server wrote them", () => {
        expect(fill("1st place wins {battle.points.first} points, 2nd {battle.points.second}", urf))
            .toEqual({text: "1st place wins 75 points, 2nd 54", problems: []})
        expect(fill("×{battle.times} points", urf).text).toBe("×3 points")
        expect(fill("3rd {battle.points.third}", urf)).toEqual({
            text: "3rd {battle.points.third}", problems: ["The battle has no value at {battle.points.third}."],
        })
        expect(fill("{battle.points.fourth}", urf).problems)
            .toEqual(["{battle.points.fourth} names no number or duration of the rules or the battle."])
    })
})

describe("namesBattle", () => {
    it("is whether the words name a value of the battle", () => {
        expect(namesBattle("1st place wins {battle.points.first} points", TRIPLE)).toBe(true)
        expect(namesBattle("Clicks refill {clicks.tempo}× faster", TRIPLE)).toBe(false)
        expect(namesBattle("{battle.points.fourth}", TRIPLE)).toBe(false)
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
