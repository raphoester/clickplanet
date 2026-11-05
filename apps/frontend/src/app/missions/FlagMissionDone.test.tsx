// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {FlagMission} from "../../backends/missions.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import FlagMissionDone from "./FlagMissionDone.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

const mission: FlagMission = {
    id: "flag-22",
    order: {kind: "breakShields", country: "pw", goal: 450, reached: 450},
    points: 2,
    done: true,
}

describe("FlagMissionDone", () => {
    it("cheers the flag, says what it did and the season points it won", () => {
        render(<FlagMissionDone mission={mission} country="fr" play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Congratulations! France"})).toBeDefined()
        expect(screen.getByText(/Flag mission done: Break 450 shields of/)).toBeDefined()
        expect(screen.getByText("+2 season points")).toBeDefined()
    })

    it("plays the fanfare as it opens", () => {
        const play = vi.fn()
        render(<FlagMissionDone mission={mission} country="fr" play={play} onClose={vi.fn()}/>)

        expect(play).toHaveBeenCalledWith("title")
    })

    it("closes once the reveal has played", async () => {
        const onClose = vi.fn()
        render(<FlagMissionDone mission={mission} country="fr" play={vi.fn()} onClose={onClose}/>)
        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()

        act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "Close"}))

        expect(onClose).toHaveBeenCalledTimes(1)
    })
})
