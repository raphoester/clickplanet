// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import MissionsPlace from "./MissionsPlace.tsx"
import {DayMissions, Mission, MissionError} from "../../backends/missions.ts"

const now = Date.UTC(2026, 9, 26, 12, 0)

const mission = (overrides: Partial<Mission>): Mission => ({
    slot: 1,
    kind: "takeFor",
    target: "fr",
    goal: 60,
    progress: 0,
    collected: false,
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
            mission({slot: 1, progress: 37}),
            mission({slot: 2, kind: "takeFrom", target: "dz", goal: 40, progress: 3, reward: {kind: "spreadClicks", amount: 4}}),
            mission({slot: 3, kind: "takeIn", target: "br", goal: 200, reward: {kind: "bomb", amount: 1}}),
        )} onCollect={vi.fn()}/>)

        expect(screen.getByText("Ends in 9h 00m")).toBeDefined()
        expect(screen.getByText(/Take 60 tiles for/).textContent).toContain("France")
        expect(screen.getByText(/Take 40 tiles from/).textContent).toContain("Algeria")
        expect(screen.getByText(/Take 200 tiles in/).textContent).toContain("Brazil")
        expect(screen.getByText("37/60")).toBeDefined()
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("collects a mission that is done", async () => {
        const onCollect = vi.fn(async () => {})
        render(<MissionsPlace state={day(mission({slot: 2, progress: 60}))} onCollect={onCollect}/>)

        await act(async () => fireEvent.click(screen.getByRole("button", {name: "Collect"})))

        expect(onCollect).toHaveBeenCalledWith(2)
    })

    it("says a stack is full when the reward did not fit", async () => {
        const onCollect = vi.fn(async () => {
            throw new MissionError("full")
        })
        render(<MissionsPlace state={day(mission({progress: 60}))} onCollect={onCollect}/>)

        await act(async () => fireEvent.click(screen.getByRole("button", {name: "Collect"})))

        expect(screen.getByRole("button", {name: "Full"})).toBeDefined()
    })

    it("asks a guest to sign in to collect", () => {
        const onSignIn = vi.fn()
        const onCollect = vi.fn()
        render(<MissionsPlace state={day(mission({progress: 60}))} onCollect={onCollect} onSignIn={onSignIn}/>)

        fireEvent.click(screen.getByRole("button", {name: "Sign in to collect"}))

        expect(onSignIn).toHaveBeenCalled()
        expect(onCollect).not.toHaveBeenCalled()
    })

    it("shows a mission collected as done", () => {
        render(<MissionsPlace state={day(mission({progress: 60, collected: true}))} onCollect={vi.fn()}/>)

        expect(screen.getByText("Collected")).toBeDefined()
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("says so when the missions could not be read", () => {
        render(<MissionsPlace state={{kind: "failed"}} onCollect={vi.fn()}/>)

        expect(screen.getByRole("alert").textContent).toBe("Your missions could not be loaded.")
    })
})
