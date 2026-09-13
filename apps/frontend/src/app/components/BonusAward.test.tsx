// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import BonusAward, {AWARD_MS, DISMISS_GRACE_MS} from "./BonusAward.tsx"

const REWARD = {kind: "tripleClicks", seconds: 60} as const

describe("BonusAward", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("shouts what the player just won", () => {
        render(<BonusAward reward={REWARD} onDone={() => {}}/>)

        expect(screen.getByText("Triple clicks")).toBeTruthy()
        expect(screen.getByText(/60 seconds/)).toBeTruthy()
    })

    it("is a status, so a screen reader hears it without the focus moving", () => {
        render(<BonusAward reward={REWARD} onDone={() => {}}/>)

        expect(screen.getByRole("status")).toBeTruthy()
    })

    it("takes itself away if nobody dismisses it", () => {
        const onDone = vi.fn()
        render(<BonusAward reward={REWARD} onDone={onDone}/>)

        expect(onDone).not.toHaveBeenCalled()

        act(() => void vi.advanceTimersByTime(AWARD_MS))

        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it("still goes away while its parent re-renders around it", () => {
        // The bug this pins: the dismissal is rebuilt on every render of
        // useGlobe, and the leaderboard republishes twice a second. With the
        // callback in the effect's dependencies the timer was cleared and
        // restarted on every one of those renders, and it never left.
        const onDone = vi.fn()
        const {rerender} = render(<BonusAward reward={REWARD} onDone={onDone}/>)

        for (let tick = 0; tick * 500 < AWARD_MS; tick++) {
            act(() => void vi.advanceTimersByTime(500))
            rerender(<BonusAward reward={REWARD} onDone={() => onDone()}/>)
        }

        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it("calls the newest callback it was given, not the one it started with", () => {
        const stale = vi.fn()
        const fresh = vi.fn()
        const {rerender} = render(<BonusAward reward={REWARD} onDone={stale}/>)

        rerender(<BonusAward reward={REWARD} onDone={fresh}/>)
        act(() => void vi.advanceTimersByTime(AWARD_MS))

        expect(stale).not.toHaveBeenCalled()
        expect(fresh).toHaveBeenCalledTimes(1)
    })

    it("restarts for a second box rather than inheriting the first one's countdown", () => {
        const onDone = vi.fn()
        const {rerender} = render(<BonusAward reward={REWARD} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(AWARD_MS - 100))
        rerender(<BonusAward reward={{kind: "tripleClicks", seconds: 30}} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(200))
        expect(onDone).not.toHaveBeenCalled()

        act(() => void vi.advanceTimersByTime(AWARD_MS))
        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it("goes away on a press anywhere, once it has been up long enough to be seen", () => {
        const onDone = vi.fn()
        render(<BonusAward reward={REWARD} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(DISMISS_GRACE_MS))
        fireEvent.pointerDown(document.body)

        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it("goes away on a press on the card itself", () => {
        const onDone = vi.fn()
        render(<BonusAward reward={REWARD} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(DISMISS_GRACE_MS))
        fireEvent.pointerDown(screen.getByText("Triple clicks"))

        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it("ignores the rest of the burst that caught the box", () => {
        const onDone = vi.fn()
        render(<BonusAward reward={REWARD} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(DISMISS_GRACE_MS - 100))
        fireEvent.pointerDown(document.body)

        expect(onDone).not.toHaveBeenCalled()
    })

    it("wears a different mark and colour for each kind of bonus", () => {
        const rewards = [
            {kind: "tripleClicks", seconds: 60},
            {kind: "spreadClicks", seconds: 60},
            {kind: "bomb", seconds: 60, radius: 0.1},
            {kind: "encloseClicks", seconds: 60, shapes: 2, maxTiles: 200},
        ] as const

        const looks = rewards.map(reward => {
            const {container} = render(<BonusAward reward={reward} onDone={() => {}}/>)
            const look = {
                className: container.querySelector(".bonus-award")!.className,
                icon: container.querySelector(".bonus-award-box")!.textContent,
            }
            cleanup()
            return look
        })

        expect(new Set(looks.map(look => look.className)).size).toBe(rewards.length)
        expect(new Set(looks.map(look => look.icon)).size).toBe(rewards.length)
    })

    it("drops its timer when it goes, so nothing fires into an unmounted award", () => {
        const onDone = vi.fn()
        const {unmount} = render(<BonusAward reward={REWARD} onDone={onDone}/>)

        unmount()
        act(() => void vi.advanceTimersByTime(AWARD_MS * 2))

        expect(onDone).not.toHaveBeenCalled()
    })
})
