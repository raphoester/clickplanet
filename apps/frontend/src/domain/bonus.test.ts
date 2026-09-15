import {describe, expect, it} from "vitest"
import {ActiveBonus, afterShapeClosed, describeReward, hasLapsed, multiplierOf, secondsLeft} from "./bonus.ts"

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
        expect(detail).toBe("3× your click rate")
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
    const SPREAD = {kind: "spreadClicks", seconds: 60} as const

    it("multiplies nothing, so the meter keeps the server's plain allowance", () => {
        expect(multiplierOf(SPREAD)).toBe(1)
    })

    it("says what it does, with a badge that fits the meter", () => {
        const {title, detail, badge} = describeReward(SPREAD)

        expect(title).toBe("Spread clicks")
        expect(detail).toBe("Each click also takes the tiles around it")
        expect(badge.length).toBeLessThanOrEqual(3)
    })
})

describe("a bomb", () => {
    const BOMB = {kind: "bomb", seconds: 30, radius: 0.06} as const

    it("multiplies nothing", () => {
        expect(multiplierOf(BOMB)).toBe(1)
    })

    it("says what it does, with a badge that fits the meter", () => {
        const {title, detail, badge} = describeReward(BOMB)

        expect(title).toBe("Bomb")
        expect(detail).toBe("Resets the tiles in an area")
        expect(badge.length).toBeLessThanOrEqual(3)
    })
})

describe("an enclose reward", () => {
    const ENCLOSE = {kind: "encloseClicks", seconds: 30, shapes: 3, maxTiles: 10} as const

    it("multiplies nothing, so the meter keeps the server's plain allowance", () => {
        expect(multiplierOf(ENCLOSE)).toBe(1)
    })

    it("says what it does, how many shapes and how big", () => {
        const {title, detail, badge} = describeReward(ENCLOSE)

        expect(title).toBe("Enclose")
        expect(detail).toContain("3 shapes")
        expect(detail).toContain("10 tiles")
        expect(detail).not.toContain("30")
        expect(badge.length).toBeLessThanOrEqual(3)
    })

    it("counts the shapes left down on the badge", () => {
        const running: ActiveBonus = {reward: ENCLOSE, endsAt: 30_000}

        const after = afterShapeClosed(running, 2)

        expect(after?.reward).toMatchObject({shapes: 2})
        expect(after?.endsAt).toBe(30_000)
        expect(describeReward(after!.reward).badge).toContain("2")
    })

    it("is over once its last shape is closed, whatever the clock says", () => {
        expect(afterShapeClosed({reward: ENCLOSE, endsAt: 30_000}, 0)).toBeUndefined()
    })

    it("leaves any other bonus alone", () => {
        expect(afterShapeClosed(RUNNING, 0)).toBe(RUNNING)
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
