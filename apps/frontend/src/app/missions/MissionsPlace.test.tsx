// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import MissionsPlace from "./MissionsPlace.tsx"
import {DayMissions, Mission} from "../../backends/missions.ts"

const now = Date.UTC(2026, 9, 26, 12, 0)

const mission = (overrides: Partial<Mission>): Mission => ({
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 0},
    reward: {kind: "shields", amount: 10},
    done: false,
    paid: false,
    ...overrides,
})

const doneBank: Partial<Mission> = {order: {kind: "takeFor", country: "fr", goal: 60, taken: 60}, done: true}

const day = (...missions: Mission[]): {kind: "ready", day: DayMissions} => ({
    kind: "ready",
    day: {endsAt: Date.UTC(2026, 9, 26, 21, 0), missions},
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("MissionsPlace", () => {
    it("lists each mission with its country, how far it went and when the day ends", () => {
        vi.useFakeTimers({now})
        render(<MissionsPlace state={day(
            mission({id: "bank", order: {kind: "takeFor", country: "fr", goal: 60, taken: 37}}),
            mission({id: "rival", order: {kind: "takeFrom", country: "dz", goal: 40, taken: 3}, reward: {kind: "spreadClicks", amount: 4}}),
            mission({id: "ground", order: {kind: "takeIn", country: "br", goal: 200, taken: 0}, reward: {kind: "bomb", amount: 1}}),
        )}/>)

        expect(screen.getByText("Ends in 9h 00m")).toBeDefined()
        expect(screen.getByText(/Take 60 tiles for/).textContent).toContain("France")
        expect(screen.getByText(/Take 40 tiles from/).textContent).toContain("Algeria")
        expect(screen.getByText(/Take 200 tiles in/).textContent).toContain("Brazil")
        expect(screen.getByText("37/60")).toBeDefined()
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("shows the flag's mission above the player's, with the points it gives the flag", () => {
        render(<MissionsPlace state={{kind: "ready", day: {
            ...day(mission({})).day,
            flagMission: {id: "flag", order: {kind: "breakShields", country: "dz", goal: 300, reached: 120}, points: 2, done: false},
        }}}/>)

        const flag = screen.getByRole("region", {name: "Flag mission"})
        expect(flag.textContent).toContain("Break 300 shields of")
        expect(flag.textContent).toContain("Algeria")
        expect(flag.textContent).toContain("+2 points")
        expect(screen.getByText("120/300")).toBeDefined()
    })

    it("names both countries of a raid, and counts what a feat asks", () => {
        render(<MissionsPlace state={day(
            mission({id: "raid", order: {kind: "takeFromIn", country: "dz", ground: "br", goal: 60, taken: 7}}),
            mission({id: "bomb", order: {kind: "bombIn", country: "br", goal: 2, reached: 1}}),
            mission({id: "boxes", order: {kind: "catchBoxes", goal: 4, reached: 3}}),
        )}/>)

        expect(screen.getByText(/Take 60 tiles from/).textContent).toMatch(/Algeria in .*Brazil/)
        expect(screen.getByText(/Drop 2 bombs on/).textContent).toContain("Brazil")
        expect(screen.getByText("Catch 4 boxes")).toBeDefined()
        expect(screen.getByText("3/4")).toBeDefined()
    })

    it("asks a guest to sign in to get a mission it finished", () => {
        const onSignIn = vi.fn()
        render(<MissionsPlace state={day(mission(doneBank))} onSignIn={onSignIn}/>)

        fireEvent.click(screen.getByRole("button", {name: "Sign in to get it"}))

        expect(onSignIn).toHaveBeenCalled()
    })

    it("asks a guest with no mission to sign in", () => {
        const onSignIn = vi.fn()
        render(<MissionsPlace state={day()} onSignIn={onSignIn}/>)

        expect(screen.getByText("Sign in to get missions.")).toBeDefined()
        fireEvent.click(screen.getByRole("button", {name: "Sign in"}))

        expect(onSignIn).toHaveBeenCalled()
    })

    it("says so when a signed-in player has no mission", () => {
        render(<MissionsPlace state={day()}/>)

        expect(screen.getByText("No missions right now.")).toBeDefined()
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("shows a mission done as done, with nothing to press", () => {
        render(<MissionsPlace state={day(mission({...doneBank, paid: true}), mission({...doneBank, id: "rival"}))}/>)

        expect(screen.getAllByText("Done")).toHaveLength(2)
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("tells what it showed, so the badge forgets it", () => {
        const onSeen = vi.fn()
        const shown = day(mission({}), mission({...doneBank, id: "rival"}))

        render(<MissionsPlace state={{kind: "loading"}} onSeen={onSeen}/>)
        expect(onSeen).not.toHaveBeenCalled()
        cleanup()
        render(<MissionsPlace state={shown} onSeen={onSeen}/>)

        expect(onSeen).toHaveBeenCalledWith(shown.day.missions)
    })

    it("says so when the missions could not be read", () => {
        render(<MissionsPlace state={{kind: "failed"}}/>)

        expect(screen.getByRole("alert").textContent).toBe("Your missions could not be loaded.")
    })
})
