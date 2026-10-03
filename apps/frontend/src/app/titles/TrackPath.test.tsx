// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import {PlayerTitle, TitleTrack} from "../../backends/player.ts"
import TrackPath from "./TrackPath.tsx"
import {filledOf} from "./titleArt.ts"

afterEach(cleanup)

const rank = (id: string, name: string, number: number): PlayerTitle =>
    ({id, name, rank: {trackId: "conquest", trackName: "Conquest", number, count: 3}})

const conquest = (progress: number, earned: number): TitleTrack => ({
    id: "conquest", name: "Conquest", progress, steps: [
        {title: rank("settler", "Settler", 1), threshold: 100, earned: earned >= 1},
        {title: rank("raider", "Raider", 2), threshold: 1_000, earned: earned >= 2},
        {title: rank("warlord", "Warlord", 3), threshold: 10_000, earned: earned >= 3},
    ],
})

describe("filledOf", () => {
    it("fills nothing before the first rank", () => {
        expect(filledOf(conquest(40, 0))).toBe(0)
    })

    it("fills up to the last rank held, then the share of the way to the next", () => {
        expect(filledOf(conquest(550, 1))).toBeCloseTo(0.25)
        expect(filledOf(conquest(1_000, 2))).toBeCloseTo(0.5)
    })

    it("fills the whole bar once every rank is held", () => {
        expect(filledOf(conquest(12_000, 3))).toBe(1)
    })

    it("goes no further than the next rank, whatever the progress says", () => {
        expect(filledOf(conquest(5_000, 1))).toBeCloseTo(0.5)
    })
})

describe("TrackPath", () => {
    it("says how far the next rank is, and marks it as the current step", () => {
        render(<TrackPath track={conquest(140, 1)}/>)

        const track = screen.getByRole("region", {name: "Conquest"})
        expect(within(track).getByText("860 tiles to Raider")).toBeDefined()
        const steps = within(track).getAllByRole("listitem")
        expect(steps.map((step) => step.getAttribute("aria-current"))).toEqual([null, "step", null])
        expect(within(steps[2]).getByText((10_000).toLocaleString() + " tiles")).toBeDefined()
    })

    it("counts a streak in days", () => {
        render(<TrackPath track={{
            id: "devotion", name: "Devotion", progress: 12, steps: [
                {title: {id: "loyal", name: "Loyal"}, threshold: 7, earned: true},
                {title: {id: "devoted", name: "Devoted"}, threshold: 30, earned: false},
            ],
        }}/>)

        expect(screen.getByText("Day 12 · 18 more to Devoted")).toBeDefined()
        expect(screen.getByText("30 days in a row")).toBeDefined()
    })

    it("counts the chat in messages", () => {
        render(<TrackPath track={{
            id: "chatter", name: "Chatter", progress: 140, steps: [
                {title: {id: "talker", name: "Talker"}, threshold: 100, earned: true},
                {title: {id: "chatterbox", name: "Chatterbox"}, threshold: 1_000, earned: false},
            ],
        }}/>)

        expect(screen.getByText("860 messages to Chatterbox")).toBeDefined()
        expect(screen.getByText((1_000).toLocaleString() + " messages")).toBeDefined()
    })

    it("says nothing is left once every rank is held", () => {
        render(<TrackPath track={conquest(12_000, 3)}/>)

        expect(screen.queryByText(/ to /)).toBeNull()
        expect(screen.getAllByRole("listitem").every((step) => step.getAttribute("aria-current") === null)).toBe(true)
    })
})
