// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {PlayerTitle, TitleTrack} from "../../backends/progression.ts"
import ProfileTracks from "./ProfileTracks.tsx"

afterEach(cleanup)

const rank = (id: string, name: string): PlayerTitle => ({id, name})

const conquest: TitleTrack = {
    id: "conquest", name: "Conquest", progress: 140, steps: [
        {title: rank("settler", "Settler"), threshold: 100, earned: true},
        {title: rank("raider", "Raider"), threshold: 1_000, earned: false},
        {title: rank("warlord", "Warlord"), threshold: 10_000, earned: false},
    ],
}

const devotion: TitleTrack = {
    id: "devotion", name: "Devotion", progress: 12, steps: [
        {title: rank("loyal", "Loyal"), threshold: 7, earned: true},
        {title: rank("devoted", "Devoted"), threshold: 30, earned: false},
    ],
}

describe("ProfileTracks", () => {
    it("shows the first track's ranks until another is picked", () => {
        render(<ProfileTracks tracks={[conquest, devotion]}/>)

        const tabs = screen.getAllByRole("tab")
        expect(tabs.map((tab) => tab.getAttribute("aria-selected"))).toEqual(["true", "false"])
        const ranks = screen.getByRole("tabpanel")
        expect(within(ranks).getByText("860 tiles to Raider")).toBeDefined()
        const steps = within(ranks).getAllByRole("listitem")
        expect(steps.map((step) => step.getAttribute("aria-current"))).toEqual([null, "step", null])
    })

    it("shows every rank of the track picked, beside the list", () => {
        render(<ProfileTracks tracks={[conquest, devotion]}/>)

        fireEvent.click(screen.getByRole("tab", {name: /Devotion/}))

        expect(screen.getByRole("tab", {name: /Devotion/}).getAttribute("aria-selected")).toBe("true")
        const ranks = screen.getByRole("tabpanel", {name: /Devotion/})
        expect(within(ranks).getByText("Day 12 · 18 more to Devoted")).toBeDefined()
        expect(within(ranks).getByText("30 days in a row")).toBeDefined()
    })
})
