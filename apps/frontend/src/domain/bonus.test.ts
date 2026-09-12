import {describe, expect, it} from "vitest"
import {ActiveBonus, describeReward, hasLapsed, multiplierOf, secondsLeft} from "./bonus.ts"

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
        expect(detail).toContain("60")
        expect(badge).toBe("3×")
    })

    it("takes the duration from the reward rather than stating one", () => {
        expect(describeReward({kind: "tripleClicks", seconds: 5}).detail).toContain("5")
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
    const SPREAD = {kind: "spreadClicks", seconds: 60} as const

    it("multiplies nothing, so the meter keeps the server's plain allowance", () => {
        expect(multiplierOf(SPREAD)).toBe(1)
    })

    it("says what it does, for how long, with a badge that fits the meter", () => {
        const {title, detail, badge} = describeReward(SPREAD)

        expect(title).toBe("Spread clicks")
        expect(detail).toContain("60")
        expect(badge.length).toBeLessThanOrEqual(3)
    })
})

describe("a bomb", () => {
    const BOMB = {kind: "bomb", seconds: 30, radius: 0.06} as const

    it("multiplies nothing", () => {
        expect(multiplierOf(BOMB)).toBe(1)
    })

    it("says how long there is to drop it, with a badge that fits the meter", () => {
        const {title, detail, badge} = describeReward(BOMB)

        expect(title).toBe("Bomb")
        expect(detail).toContain("30")
        expect(badge.length).toBeLessThanOrEqual(3)
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
