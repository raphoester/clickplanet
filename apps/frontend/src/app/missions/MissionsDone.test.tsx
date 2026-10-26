// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import MissionsDone from "./MissionsDone.tsx"
import {Mission} from "../../backends/missions.ts"
import {AWARD_MS} from "../components/BonusAward.tsx"

const rules = {blastRadius: 0.05, enclosureMaxTiles: 25}

const bank: Mission = {
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 60},
    reward: {kind: "shields", amount: 10},
    done: true,
    paid: true,
}

const ground: Mission = {
    id: "ground",
    order: {kind: "takeIn", country: "br", goal: 200, taken: 200},
    reward: {kind: "bomb", amount: 1},
    done: true,
    paid: true,
}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("MissionsDone", () => {
    it("names the mission done and what it paid", () => {
        render(<MissionsDone missions={[bank]} rules={rules} onDone={vi.fn()}/>)

        expect(screen.getByText("Mission done")).toBeDefined()
        expect(screen.getByText(/Take 60 tiles for/).textContent).toContain("France")
        expect(screen.getByText("+10 shields")).toBeDefined()
    })

    it("tells every mission done at once in one card", () => {
        render(<MissionsDone missions={[bank, ground]} rules={rules} onDone={vi.fn()}/>)

        expect(screen.getAllByRole("status")).toHaveLength(1)
        expect(screen.getByText("2 missions done")).toBeDefined()
        expect(screen.getByText(/Take 200 tiles in/).textContent).toContain("Brazil")
        expect(screen.getByText("Bomb")).toBeDefined()
    })

    it("goes by itself", () => {
        vi.useFakeTimers()
        const onDone = vi.fn()
        render(<MissionsDone missions={[bank]} rules={rules} onDone={onDone}/>)

        act(() => vi.advanceTimersByTime(AWARD_MS))

        expect(onDone).toHaveBeenCalledOnce()
    })
})
