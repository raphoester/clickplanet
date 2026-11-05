// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {ClosedRound, RoundStanding, Score} from "../../backends/standings.ts"
import {ROUND_REVEAL} from "../../domain/roundReveal.ts"
import RoundReveal from "./RoundReveal.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

const result = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})

const DAY: ClosedRound = {
    season: 0,
    number: 5,
    endedAt: Date.UTC(2026, 9, 16, 21),
    finale: false,
    standings: [result(1, "fr", 25), result(2, "de", 18), result(3, "es", 15)],
    before: [score(1, "de", 50), score(2, "fr", 30)],
    after: [score(1, "de", 68), score(2, "fr", 55), score(3, "es", 15)],
    moment: "day",
    times: 1,
    startedAt: 0,
    entryId: "",
    roundId: "season-0/day-5",
    takes: [],
    players: [],
}

const FINALE: ClosedRound = {...DAY, number: 23, finale: true}
const FINAL_BATTLE: ClosedRound = {...DAY, number: 1, finale: true, moment: "battle", times: 3, startedAt: Date.UTC(2026, 9, 16, 19), roundId: "entry-final"}

const user = () => userEvent.setup({advanceTimers: vi.advanceTimersByTime})

const after = (seconds: number) => act(() => vi.advanceTimersByTime(seconds * 1000))

const rowOf = (name: string) => screen.getByText(name).closest("li")!

describe("RoundReveal", () => {
    it("says the day is over and shows its podium, with the points each country won", () => {
        render(<RoundReveal closed={DAY} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)

        const dialog = screen.getByRole("dialog", {name: "Day 5 is over!"})
        expect(within(dialog).getByText("France")).toBeDefined()
        expect(within(dialog).getByText("+25")).toBeDefined()
        expect(within(dialog).getByLabelText("Rank 2")).toBeDefined()
    })

    it("names a day that ends the season by its number", () => {
        render(<RoundReveal closed={FINALE} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Day 23 is over!"})).toBeDefined()
    })

    it("plays the fanfare as it opens", () => {
        const play = vi.fn()
        render(<RoundReveal closed={DAY} countryCode="fr" play={play} onClose={vi.fn()}/>)

        expect(play).toHaveBeenCalledWith("title")
    })

    it("moves on to the season, from the order before the day to the order after it", () => {
        render(<RoundReveal closed={DAY} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)
        after(ROUND_REVEAL.podium)

        expect(screen.getByText("Season 0")).toBeDefined()
        expect(within(rowOf("France")).getByText("30")).toBeDefined()
        expect(within(rowOf("Spain")).getByText("–")).toBeDefined()

        after(ROUND_REVEAL.slide)
        after(ROUND_REVEAL.settle)

        expect(within(rowOf("France")).getByText("55")).toBeDefined()
        expect(within(rowOf("France")).getByText("+25")).toBeDefined()
        expect(within(rowOf("Spain")).getByText("New")).toBeDefined()
        expect(rowOf("France").className).toContain("round-reveal-row-mine")
    })

    it("can skip the podium", async () => {
        render(<RoundReveal closed={DAY} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)

        await user().click(screen.getByRole("button", {name: "Next"}))

        expect(screen.getByText("Season 0")).toBeDefined()
    })

    it("closes only once the table has settled", async () => {
        const onClose = vi.fn()
        render(<RoundReveal closed={DAY} countryCode="fr" play={vi.fn()} onClose={onClose}/>)

        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()
        after(ROUND_REVEAL.podium)
        after(ROUND_REVEAL.slide)
        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()
        after(ROUND_REVEAL.settle)
        await user().click(screen.getByRole("button", {name: "Close"}))

        expect(onClose).toHaveBeenCalledOnce()
    })

    it("ends the Final Battle on the season's champion", async () => {
        const onClose = vi.fn()
        render(<RoundReveal closed={FINALE} countryCode="fr" play={vi.fn()} onClose={onClose}/>)
        after(ROUND_REVEAL.podium)
        after(ROUND_REVEAL.slide + ROUND_REVEAL.settle)
        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()

        after(ROUND_REVEAL.hold)

        const dialog = screen.getByRole("dialog", {name: "Germany wins Season 0"})
        expect(within(dialog).getByText("🏆")).toBeDefined()
        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()
        after(ROUND_REVEAL.champion)
        await user().click(screen.getByRole("button", {name: "Close"}))
        expect(onClose).toHaveBeenCalledOnce()
    })

    it("plays the fanfare again for the champion", () => {
        const play = vi.fn()
        render(<RoundReveal closed={FINALE} countryCode="fr" play={play} onClose={vi.fn()}/>)
        after(ROUND_REVEAL.podium)
        after(ROUND_REVEAL.slide + ROUND_REVEAL.settle)
        after(ROUND_REVEAL.hold)

        expect(play).toHaveBeenCalledTimes(2)
    })

    it("can skip the table to the champion", async () => {
        render(<RoundReveal closed={FINALE} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)

        await user().click(screen.getByRole("button", {name: "Next"}))
        await user().click(screen.getByRole("button", {name: "Next"}))

        expect(screen.getByRole("dialog", {name: "Germany wins Season 0"})).toBeDefined()
        expect(screen.queryByRole("button", {name: "Next"})).toBeNull()
    })

    it("crowns nobody after a day", () => {
        render(<RoundReveal closed={DAY} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)
        after(ROUND_REVEAL.podium)
        after(ROUND_REVEAL.slide + ROUND_REVEAL.settle)
        after(ROUND_REVEAL.hold)

        expect(screen.queryByText("🏆")).toBeNull()
        expect(screen.getByText("Season 0")).toBeDefined()
        expect(screen.getByRole("button", {name: "Close"})).toBeDefined()
    })

    it("shows the season's results with no podium: the table before and after, then the champion", async () => {
        const onClose = vi.fn()
        render(<RoundReveal closed={FINAL_BATTLE} results countryCode="fr" play={vi.fn()} onClose={onClose}/>)

        expect(screen.getByRole("dialog", {name: "Season 0 is over!"})).toBeDefined()
        expect(screen.queryByLabelText("Rank 2")).toBeNull()
        expect(screen.getByText("Final table")).toBeDefined()
        expect(within(rowOf("France")).getByText("30")).toBeDefined()

        after(ROUND_REVEAL.slide)
        after(ROUND_REVEAL.settle)
        expect(within(rowOf("France")).getByText("55")).toBeDefined()
        after(ROUND_REVEAL.hold)

        expect(screen.getByRole("dialog", {name: "Germany wins Season 0"})).toBeDefined()
        after(ROUND_REVEAL.champion)
        await user().click(screen.getByRole("button", {name: "Close"}))
        expect(onClose).toHaveBeenCalledOnce()
    })

    it("goes straight to the season when nobody scored the day", () => {
        render(<RoundReveal closed={{...DAY, standings: []}} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByText("Season 0")).toBeDefined()
    })

    it("ends on the podium when the season has no table yet", () => {
        render(<RoundReveal closed={{...DAY, before: [], after: []}} countryCode="fr" play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: "Next"})).toBeNull()
        after(ROUND_REVEAL.podium)
        expect(screen.queryByText("Season 0")).toBeNull()
        expect(screen.getByRole("button", {name: "Close"})).toBeDefined()
    })
})
