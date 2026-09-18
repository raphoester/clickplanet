import {describe, expect, it} from "vitest"
import {ActiveBonus, chargeLabels, describeReward, hasLapsed, isTimed, multiplierOf, NO_CHARGES, secondsLeft} from "./bonus.ts"

const REWARD = {kind: "tripleClicks", seconds: 60} as const

const RUNNING: ActiveBonus = {reward: REWARD, endsAt: 60_000}

describe("multiplierOf", () => {
    it("reads the multiplier off the reward rather than restating it", () => {
        expect(multiplierOf(REWARD)).toBe(3)
    })
})

describe("describeReward", () => {
    it("gives all three lengths the screen needs", () => {
        const {title, detail, badge} = describeReward(REWARD)

        expect(title).toBe("Triple clicks")
        expect(detail).toBe("Clicks refill 3× faster")
        expect(badge).toBe("3×")
    })

    it("leaves the duration to the meter", () => {
        expect(describeReward(REWARD).detail).not.toContain("60")
    })

    it("keeps the badge short enough to sit on the meter", () => {
        expect(describeReward(REWARD).badge.length).toBeLessThanOrEqual(3)
    })

    it("says the same multiplier in the words and on the badge", () => {
        const {detail, badge} = describeReward(REWARD)

        expect(badge).toContain(String(multiplierOf(REWARD)))
        expect(detail).toContain(String(multiplierOf(REWARD)))
    })
})

describe("a spread reward", () => {
    const SPREAD = {kind: "spreadClicks", clicks: 8} as const

    it("multiplies nothing, so the meter keeps the server's plain allowance", () => {
        expect(multiplierOf(SPREAD)).toBe(1)
    })

    it("says what it does and for how many clicks, with a badge that fits the meter", () => {
        const {title, detail, badge} = describeReward(SPREAD)

        expect(title).toBe("Spread clicks")
        expect(detail).toBe("Your next 8 clicks also take the tiles around them")
        expect(badge.length).toBeLessThanOrEqual(3)
    })

    it("is a charge, not a timed bonus", () => {
        expect(isTimed(SPREAD)).toBe(false)
    })
})

describe("a bomb", () => {
    const BOMB = {kind: "bomb", radius: 0.06} as const

    it("multiplies nothing", () => {
        expect(multiplierOf(BOMB)).toBe(1)
    })

    it("says what it does and that it keeps, with a badge that fits the meter", () => {
        const {title, detail, badge} = describeReward(BOMB)

        expect(title).toBe("Bomb")
        expect(detail).toBe("Resets the tiles in an area. Kept until you drop it")
        expect(badge.length).toBeLessThanOrEqual(3)
    })
})

describe("an enclose reward", () => {
    const ENCLOSE = {kind: "encloseClicks", maxTiles: 25} as const

    it("multiplies nothing, so the meter keeps the server's plain allowance", () => {
        expect(multiplierOf(ENCLOSE)).toBe(1)
    })

    it("says what it does and how big a shape", () => {
        const {title, detail, badge} = describeReward(ENCLOSE)

        expect(title).toBe("Enclose")
        expect(detail).toContain("25 tiles")
        expect(badge.length).toBeLessThanOrEqual(3)
    })
})

describe("only a triple runs for a time", () => {
    it("tells the timed reward from the charges", () => {
        expect(isTimed(REWARD)).toBe(true)
        expect(isTimed({kind: "bomb", radius: 0.03})).toBe(false)
        expect(isTimed({kind: "encloseClicks", maxTiles: 25})).toBe(false)
    })
})

describe("chargeLabels", () => {
    it("says nothing when nothing is held", () => {
        expect(chargeLabels(NO_CHARGES)).toEqual([])
    })

    it("says each charge held, the bomb first", () => {
        expect(chargeLabels({
            bomb: true,
            enclose: true,
            spreadClicksLeft: 5,
        })).toEqual([
            {kind: "bomb", label: "Bomb ready"},
            {kind: "encloseClicks", label: "Enclose ready"},
            {kind: "spreadClicks", label: "Spread: 5 clicks left"},
        ])
    })

    it("counts the last spread click as one click", () => {
        expect(chargeLabels({...NO_CHARGES, spreadClicksLeft: 1})).toEqual([{kind: "spreadClicks", label: "Spread: 1 click left"}])
    })
})

describe("secondsLeft", () => {
    it("counts down against the clock it was stamped on", () => {
        expect(secondsLeft(RUNNING, 0)).toBe(60)
        expect(secondsLeft(RUNNING, 30_000)).toBe(30)
    })

    it("rounds up, so the last second is shown as one rather than none", () => {
        expect(secondsLeft(RUNNING, 59_400)).toBe(1)
        expect(secondsLeft(RUNNING, 59_999)).toBe(1)
    })

    it("never goes negative once it has run out", () => {
        expect(secondsLeft(RUNNING, 60_000)).toBe(0)
        expect(secondsLeft(RUNNING, 120_000)).toBe(0)
    })
})

describe("hasLapsed", () => {
    it("is running right up to the moment it ends", () => {
        expect(hasLapsed(RUNNING, 59_999)).toBe(false)
        expect(hasLapsed(RUNNING, 60_000)).toBe(true)
    })

    it("agrees with the countdown: nothing reads zero while it still runs", () => {
        for (let at = 0; at < 60_000; at += 137) {
            expect(secondsLeft(RUNNING, at)).toBeGreaterThan(0)
            expect(hasLapsed(RUNNING, at)).toBe(false)
        }
    })
})
