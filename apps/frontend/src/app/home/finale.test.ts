// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import home from "../../../index.html?raw"
import {FakeTurnBackend, SEASON_ZERO} from "../../backends/fakeTurnBackend.ts"
import {finaleShown} from "./finale.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE

const now = Date.UTC(2026, 9, 30, 18)

describe("the Final Battle on the home page", () => {
    let page: Document
    let turns: FakeTurnBackend
    let stop: () => void = () => {}

    const panel = () => page.querySelector<HTMLElement>("[data-finale]")!
    const field = (name: string) => page.querySelector(`[data-finale-${name}]`)?.textContent
    const clock = () => ["days", "hours", "minutes", "seconds"].map(field).join(":")
    const hidden = (name: string) => page.querySelector<HTMLElement>(`[data-finale-${name}]`)!.hidden

    beforeEach(() => {
        vi.useFakeTimers({now})
        page = new DOMParser().parseFromString(home, "text/html")
        turns = new FakeTurnBackend()
    })

    const follow = () => {
        const finale = finaleShown(page)
        const unfollow = turns.follow({onTurn: finale.showTurn, onBattles: finale.showBattles, onSeasons: () => {}, onDebrief: () => {}})
        return () => {
            unfollow()
            finale.stop()
        }
    }

    afterEach(() => {
        stop()
        vi.useRealTimers()
    })

    it("stays hidden while no finale is to come, even with a battle before it", () => {
        turns.scheduleBattle(now + HOUR, 30)
        stop = follow()
        expect(panel().hidden).toBe(true)
    })

    it("says when it plays, how long, what it is worth, and the operator's words", () => {
        turns.scheduleFinale(120, "Triple points. A bonus box every minute.")
        stop = follow()

        expect(panel().hidden).toBe(false)
        expect(field("when")).toMatch(/^\w{3} \d{1,2} \w{3} · \d\d:\d\d – \d\d:\d\d$/)
        expect(field("length")).toBe("2 hours")
        expect(field("times")).toBe("3")
        expect(hidden("times-row")).toBe(false)
        expect(field("words")).toBe("Triple points. A bonus box every minute.")
        expect(hidden("rules")).toBe(false)
    })

    it("counts down to its start, then, live, to its end", () => {
        turns.scheduleFinale(120)
        stop = follow()

        expect(panel().hasAttribute("data-live")).toBe(false)
        expect(clock()).toBe("01:03:00:00")

        vi.advanceTimersByTime(SECOND)
        expect(clock()).toBe("01:02:59:59")

        vi.setSystemTime(SEASON_ZERO.endsAt - 2 * HOUR)
        vi.advanceTimersByTime(SECOND)
        expect(panel().hasAttribute("data-live")).toBe(true)
        expect(clock()).toBe("00:01:59:59")
    })

    it("follows the turn that plays it, and hides once it ended", () => {
        stop = follow()
        turns.startFinale(now, "", 1)

        expect(panel().hidden).toBe(false)
        expect(panel().hasAttribute("data-live")).toBe(true)
        expect(hidden("rules")).toBe(true)
        expect(hidden("times-row")).toBe(true)

        turns.endBattle()
        expect(panel().hidden).toBe(true)
    })
})
