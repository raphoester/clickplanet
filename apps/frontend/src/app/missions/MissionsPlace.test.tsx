// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import MissionsPlace from "./MissionsPlace.tsx"
import {DayMissions, Mission} from "../../backends/missions.ts"

const now = Date.UTC(2026, 9, 26, 12, 0)

const mission = (overrides: Partial<Mission>): Mission => ({
    id: "bank",
    kind: "takeFor",
    target: "fr",
    goal: 60,
    progress: 0,
    paid: false,
    reward: {kind: "shields", amount: 10},
    ...overrides,
})

const day = (...missions: Mission[]): {kind: "ready", day: DayMissions} => ({
    kind: "ready",
    day: {countryCode: "fr", enlisted: true, endsAt: Date.UTC(2026, 9, 26, 21, 0), missions},
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("MissionsPlace", () => {
    it("lists each mission with its country, how far it went and when the day ends", () => {
        vi.useFakeTimers({now})
        render(<MissionsPlace state={day(
            mission({id: "bank", progress: 37}),
            mission({id: "rival", kind: "takeFrom", target: "dz", goal: 40, progress: 3, reward: {kind: "spreadClicks", amount: 4}}),
            mission({id: "ground", kind: "takeIn", target: "br", goal: 200, reward: {kind: "bomb", amount: 1}}),
        )}/>)

        expect(screen.getByText("Ends in 9h 00m")).toBeDefined()
        expect(screen.getByText(/Take 60 tiles for/).textContent).toContain("France")
        expect(screen.getByText(/Take 40 tiles from/).textContent).toContain("Algeria")
        expect(screen.getByText(/Take 200 tiles in/).textContent).toContain("Brazil")
        expect(screen.getByText("37/60")).toBeDefined()
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("asks a guest to sign in to get a mission it finished", () => {
        const onSignIn = vi.fn()
        render(<MissionsPlace state={day(mission({progress: 60}))} onSignIn={onSignIn}/>)

        fireEvent.click(screen.getByRole("button", {name: "Sign in to get it"}))

        expect(onSignIn).toHaveBeenCalled()
    })

    it("shows a mission done as done, with nothing to press", () => {
        render(<MissionsPlace state={day(mission({progress: 60, paid: true}), mission({id: "rival", progress: 60}))}/>)

        expect(screen.getAllByText("Done")).toHaveLength(2)
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("says so when the missions could not be read", () => {
        render(<MissionsPlace state={{kind: "failed"}}/>)

        expect(screen.getByRole("alert").textContent).toBe("Your missions could not be loaded.")
    })
})
