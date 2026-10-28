// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import home from "../../../index.html?raw"
import {FakeTurnBackend} from "../../backends/fakeTurnBackend.ts"
import {nextBattleShown} from "./battle.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const now = Date.UTC(2026, 9, 27, 18)

describe("the next battle on the home page", () => {
    let page: Document
    let turns: FakeTurnBackend
    let stop: () => void = () => {}

    const pill = () => page.querySelector<HTMLElement>("[data-battle]")!
    const left = () => page.querySelector("[data-battle-left]")?.textContent

    beforeEach(() => {
        vi.useFakeTimers({now})
        page = new DOMParser().parseFromString(home, "text/html")
        turns = new FakeTurnBackend()
    })

    const follow = () => {
        const battle = nextBattleShown(page)
        const unfollow = turns.follow({onTurn: () => {}, onBattles: battle.show, onSeasons: () => {}, onDebrief: () => {}})
        return () => {
            unfollow()
            battle.stop()
        }
    }

    afterEach(() => {
        stop()
        vi.useRealTimers()
    })

    it("stays hidden while no battle is to come", () => {
        stop = follow()
        expect(pill().hidden).toBe(true)
    })

    it("counts down to the next battle's start, every second", () => {
        turns.scheduleBattle(now + 4 * DAY + 3 * HOUR, 120)
        stop = follow()

        expect(pill().hidden).toBe(false)
        expect(left()).toBe("4d 03h 00m 00s")

        vi.advanceTimersByTime(SECOND)
        expect(left()).toBe("4d 02h 59m 59s")
    })

    it("follows a battle moved, added before it, and deleted", () => {
        const finale = turns.scheduleBattle(now + DAY, 120)
        stop = follow()

        turns.moveBattle(finale, now + 2 * DAY)
        expect(left()).toBe("2d 00h 00m 00s")

        const sooner = turns.scheduleBattle(now + HOUR, 30)
        expect(left()).toBe("1h 00m 00s")

        turns.deleteBattle(sooner)
        turns.deleteBattle(finale)
        expect(pill().hidden).toBe(true)
    })

    it("hides once the battle starts, and counts to the one after it", () => {
        turns.scheduleBattle(now + MINUTE, 30)
        turns.scheduleBattle(now + DAY, 30)
        stop = follow()

        vi.advanceTimersByTime(MINUTE)
        expect(left()).toBe("23h 59m 00s")
    })
})
