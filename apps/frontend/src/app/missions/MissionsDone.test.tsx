// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import MissionsDone from "./MissionsDone.tsx"
import {Mission, MissionRates} from "../../backends/missions.ts"
import {AWARD_MS} from "../bonus/BonusAward.tsx"
import {NO_MISSIONS, useMissions} from "./missionsStore.ts"

const rates: MissionRates = {takeFor: {xp: 1, per: 1}, takeIn: {xp: 1, per: 1}}

function paid(...missions: Mission[]) {
    useMissions.setState({paid: missions, rates})
    return <MissionsDone/>
}

const bank: Mission = {
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 60},
    done: true,
    paid: true,
}

const ground: Mission = {
    id: "ground",
    order: {kind: "takeIn", country: "br", goal: 200, taken: 200},
    done: true,
    paid: true,
}

afterEach(() => {
    cleanup()
    useMissions.setState(NO_MISSIONS)
    vi.useRealTimers()
})

describe("MissionsDone", () => {
    it("names the mission done and the XP it paid", () => {
        render(paid(bank))

        expect(screen.getByText("Mission done")).toBeDefined()
        expect(screen.getByText(/Take 60 tiles for/).textContent).toContain("France")
        expect(screen.getByText("+60 XP")).toBeDefined()
    })

    it("tells every mission done at once in one card", () => {
        render(paid(bank, ground))

        expect(screen.getAllByRole("status")).toHaveLength(1)
        expect(screen.getByText("2 missions done")).toBeDefined()
        expect(screen.getByText(/Take 200 tiles in/).textContent).toContain("Brazil")
        expect(screen.getByText("+200 XP")).toBeDefined()
    })

    it("names the mission done with no XP while the rates are not read", () => {
        useMissions.setState({paid: [bank]})
        render(<MissionsDone/>)

        expect(screen.getByText("Mission done")).toBeDefined()
        expect(screen.queryByText(/XP/)).toBeNull()
    })

    it("goes by itself", () => {
        vi.useFakeTimers()
        render(paid(bank))

        act(() => vi.advanceTimersByTime(AWARD_MS))

        expect(useMissions.getState().paid).toBeUndefined()
        expect(screen.queryByRole("status")).toBeNull()
    })
})
