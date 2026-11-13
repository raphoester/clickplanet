// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import MissionsDone from "./MissionsDone.tsx"
import {Mission} from "../../backends/missions.ts"
import {AWARD_MS} from "../bonus/BonusAward.tsx"
import {useBonus} from "../bonus/bonusStore.ts"
import {NO_MISSIONS, useMissions} from "./missionsStore.ts"

const rules = {blastRadius: 0.05, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3, shields: 12, tileShields: 10, toll: [], signInRefill: false, enclosuresUnlimited: false, reach: false, knockOut: false, capitalShieldsFixed: false, bombsOff: false, spreadUnlimited: false}

function paid(...missions: Mission[]) {
    useBonus.setState({rules})
    useMissions.setState({paid: missions})
    return <MissionsDone/>
}

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
    useBonus.setState(useBonus.getInitialState(), true)
    useMissions.setState(NO_MISSIONS)
    vi.useRealTimers()
})

describe("MissionsDone", () => {
    it("names the mission done and what it paid", () => {
        render(paid(bank))

        expect(screen.getByText("Mission done")).toBeDefined()
        expect(screen.getByText(/Take 60 tiles for/).textContent).toContain("France")
        expect(screen.getByText("+10 shields")).toBeDefined()
    })

    it("tells every mission done at once in one card", () => {
        render(paid(bank, ground))

        expect(screen.getAllByRole("status")).toHaveLength(1)
        expect(screen.getByText("2 missions done")).toBeDefined()
        expect(screen.getByText(/Take 200 tiles in/).textContent).toContain("Brazil")
        expect(screen.getByText("Bomb")).toBeDefined()
    })

    it("goes by itself", () => {
        vi.useFakeTimers()
        render(paid(bank))

        act(() => vi.advanceTimersByTime(AWARD_MS))

        expect(useMissions.getState().paid).toBeUndefined()
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("waits for the rules to name what a mission paid", () => {
        useMissions.setState({paid: [bank]})
        render(<MissionsDone/>)

        expect(screen.queryByRole("status")).toBeNull()
    })
})
