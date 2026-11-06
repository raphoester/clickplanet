// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import BonusAward, {AWARD_MS, DISMISS_GRACE_MS} from "./BonusAward.tsx"
import {useBonus} from "./bonusStore.ts"
import {useBonusGuide} from "./bonusGuideStore.ts"
import {BonusReward} from "../../domain/bonus.ts"
import {BonusGuide, FRESH_GUIDE, isFirstWin} from "../../domain/bonusGuide.ts"

const REWARD = {kind: "refill"} as const

const WON_ALL: BonusGuide = {won: ["refill", "bomb", "spreadClicks", "encloseClicks", "shields"], uses: {}}

function award(reward: BonusReward, guide = WON_ALL) {
    act(() => {
        useBonusGuide.setState({guide})
        useBonus.getState().win(reward)
    })
}

const shown = () => useBonus.getState().award

describe("BonusAward", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
        useBonus.setState(useBonus.getInitialState(), true)
        useBonusGuide.setState({guide: FRESH_GUIDE})
        window.localStorage.clear()
    })

    it("shouts what the player just won", () => {
        render(<BonusAward/>)
        award(REWARD)

        expect(screen.getByText("Refill")).toBeTruthy()
        expect(screen.getByText("Fills your clicks to full, when you choose")).toBeTruthy()
    })

    it("is a status, so a screen reader hears it without the focus moving", () => {
        render(<BonusAward/>)
        award(REWARD)

        expect(screen.getByRole("status")).toBeTruthy()
    })

    it("takes itself away if nobody dismisses it", () => {
        render(<BonusAward/>)
        award(REWARD)

        act(() => void vi.advanceTimersByTime(AWARD_MS - 1))
        expect(shown()).toBe(REWARD)

        act(() => void vi.advanceTimersByTime(1))
        expect(shown()).toBeUndefined()
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("still goes away while its parent re-renders around it", () => {
        const {rerender} = render(<BonusAward/>)
        award(REWARD)

        for (let tick = 0; tick * 500 < AWARD_MS; tick++) {
            act(() => void vi.advanceTimersByTime(500))
            rerender(<BonusAward/>)
        }

        expect(shown()).toBeUndefined()
    })

    it("restarts for a second box rather than inheriting the first one's countdown", () => {
        render(<BonusAward/>)
        award(REWARD)

        act(() => void vi.advanceTimersByTime(AWARD_MS - 100))
        award({kind: "refill"})

        act(() => void vi.advanceTimersByTime(200))
        expect(shown()).toBeDefined()

        act(() => void vi.advanceTimersByTime(AWARD_MS))
        expect(shown()).toBeUndefined()
    })

    it("goes away on a press anywhere, once it has been up long enough to be seen", () => {
        render(<BonusAward/>)
        award(REWARD)

        act(() => void vi.advanceTimersByTime(DISMISS_GRACE_MS))
        fireEvent.pointerDown(document.body)

        expect(shown()).toBeUndefined()
    })

    it("goes away on a press on the card itself", () => {
        render(<BonusAward/>)
        award(REWARD)

        act(() => void vi.advanceTimersByTime(DISMISS_GRACE_MS))
        fireEvent.pointerDown(screen.getByText("Refill"))

        expect(shown()).toBeUndefined()
    })

    it("ignores the rest of the burst that caught the box", () => {
        render(<BonusAward/>)
        award(REWARD)

        act(() => void vi.advanceTimersByTime(DISMISS_GRACE_MS - 100))
        fireEvent.pointerDown(document.body)

        expect(shown()).toBe(REWARD)
    })

    it("wears a different mark and colour for each kind of bonus", () => {
        const rewards = [
            {kind: "refill"},
            {kind: "spreadClicks", clicks: 8},
            {kind: "bomb", radius: 0.1},
            {kind: "encloseClicks", shapes: 2, maxTiles: 25},
            {kind: "shields", shields: 3},
        ] as const

        const looks = rewards.map(reward => {
            const {container} = render(<BonusAward/>)
            award(reward)
            const look = {
                className: container.querySelector(".bonus-award")!.className,
                icon: container.querySelector(".bonus-award-box svg")!.innerHTML,
            }
            cleanup()
            return look
        })

        expect(new Set(looks.map(look => look.className)).size).toBe(rewards.length)
        expect(new Set(looks.map(look => look.icon)).size).toBe(rewards.length)
    })

    it("stays for a kind won for the first time, whatever is pressed beside it", () => {
        render(<BonusAward/>)
        award(REWARD, FRESH_GUIDE)

        act(() => void vi.advanceTimersByTime(AWARD_MS * 3))
        fireEvent.pointerDown(document.body)
        fireEvent.pointerDown(screen.getByText("Refill"))

        expect(shown()).toBe(REWARD)
        expect(screen.getByText("Fills your clicks to full, when you choose")).toBeTruthy()
    })

    it("goes away when a first win is read and acknowledged, and counts the kind won", () => {
        render(<BonusAward/>)
        award(REWARD, FRESH_GUIDE)

        fireEvent.click(screen.getByRole("button", {name: "Got it"}))

        expect(shown()).toBeUndefined()
        expect(isFirstWin(useBonusGuide.getState().guide, "refill")).toBe(false)
    })

    it("goes away on Escape for a first win", () => {
        render(<BonusAward/>)
        award(REWARD, FRESH_GUIDE)

        fireEvent.keyDown(document, {key: "Escape"})

        expect(shown()).toBeUndefined()
    })

    it("drops its timer when it goes, so nothing fires into an unmounted award", () => {
        const {unmount} = render(<BonusAward/>)
        award(REWARD)

        unmount()
        act(() => void vi.advanceTimersByTime(AWARD_MS * 2))

        expect(shown()).toBe(REWARD)
    })
})
