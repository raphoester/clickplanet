// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {TeaserCard} from "../../backends/promo.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import BattleTeaser from "./BattleTeaser.tsx"
import {TEASER_TURN_MS} from "./TeaserCarousel.tsx"

const now = Date.UTC(2026, 10, 5, 8)
const battle: BattleToCome = {
    entryId: "tonight", startsAt: Date.UTC(2026, 10, 5, 20), endsAt: Date.UTC(2026, 10, 5, 21),
    description: "Double points. Clicks refill 3× faster.", times: 2, endsSeason: false, roundId: "tonight",
}
const cards: TeaserCard[] = [
    {backgroundUrl: "https://media.test/war", headline: "Clicks refill 3× faster", line: "Every country fights at once."},
    {backgroundUrl: "https://media.test/points", headline: "A box every 1 min", line: ""},
    {backgroundUrl: "https://media.test/rally", headline: "Bring your friends", line: "Your flag needs every click."},
]

beforeEach(() => {
    vi.useFakeTimers({now, shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

const shownCard = () => screen.getAllByRole("figure", {hidden: true}).find((figure) => figure.getAttribute("aria-hidden") === "false")

describe("BattleTeaser", () => {
    it("says when the battle is, what it is worth and the operator's words", () => {
        render(<BattleTeaser battle={battle} cards={cards} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Battle"})).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("Starts in 12h 00m 00s")
        expect(screen.getByText(/· Points ×2$/)).toBeDefined()
        expect(screen.getByText("Double points. Clicks refill 3× faster.")).toBeDefined()
    })

    it("names the Final Battle", () => {
        render(<BattleTeaser battle={{...battle, entryId: "", endsSeason: true, roundId: "season-1/finale"}} cards={cards}
                             play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Final Battle"})).toBeDefined()
    })

    it("shows each card over its background, one at a time", () => {
        render(<BattleTeaser battle={battle} cards={cards} play={vi.fn()} onClose={vi.fn()}/>)

        const first = shownCard() as HTMLElement
        expect(first.getAttribute("aria-label")).toBe("1 of 3")
        expect(within(first).getByText("Clicks refill 3× faster")).toBeDefined()
        expect(first.querySelector("img")?.getAttribute("src")).toBe("https://media.test/war")
    })

    it("turns the cards on its own, until the player touches them", async () => {
        render(<BattleTeaser battle={battle} cards={cards} play={vi.fn()} onClose={vi.fn()}/>)

        act(() => vi.advanceTimersByTime(TEASER_TURN_MS))
        expect(shownCard()?.getAttribute("aria-label")).toBe("2 of 3")

        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "Card 1 of 3"}))
        expect(shownCard()?.getAttribute("aria-label")).toBe("1 of 3")
        act(() => vi.advanceTimersByTime(3 * TEASER_TURN_MS))
        expect(shownCard()?.getAttribute("aria-label")).toBe("1 of 3")
    })

    it("turns with the arrows, from the last card to the first", async () => {
        render(<BattleTeaser battle={battle} cards={cards} play={vi.fn()} onClose={vi.fn()}/>)
        const user = userEvent.setup({advanceTimers: vi.advanceTimersByTime})

        await user.click(screen.getByRole("button", {name: "Previous card"}))
        expect(shownCard()?.getAttribute("aria-label")).toBe("3 of 3")
        await user.click(screen.getByRole("button", {name: "Next card"}))
        expect(shownCard()?.getAttribute("aria-label")).toBe("1 of 3")
    })

    it("turns with a swipe, and stays put for a small one", () => {
        render(<BattleTeaser battle={battle} cards={cards} play={vi.fn()} onClose={vi.fn()}/>)
        const track = document.querySelector(".teaser-track") as HTMLElement
        const swipe = (from: number, to: number) => {
            fireEvent.pointerDown(track, {pointerId: 1, clientX: from})
            fireEvent.pointerMove(track, {pointerId: 1, clientX: (from + to) / 2})
            fireEvent.pointerUp(track, {pointerId: 1, clientX: to})
        }

        swipe(300, 200)
        expect(shownCard()?.getAttribute("aria-label")).toBe("2 of 3")
        swipe(200, 300)
        expect(shownCard()?.getAttribute("aria-label")).toBe("1 of 3")
        swipe(300, 280)
        expect(shownCard()?.getAttribute("aria-label")).toBe("1 of 3")
    })

    it("shows no arrow nor dot for a single card", () => {
        render(<BattleTeaser battle={battle} cards={cards.slice(0, 1)} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: "Next card"})).toBeNull()
        expect(screen.queryByRole("button", {name: /^Card 1/})).toBeNull()
    })

    it("lets the player go once it settled", async () => {
        const onClose = vi.fn()
        render(<BattleTeaser battle={battle} cards={cards} play={vi.fn()} onClose={onClose}/>)
        expect(screen.queryByRole("button", {name: "To the globe"})).toBeNull()

        act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "To the globe"}))

        expect(onClose).toHaveBeenCalledOnce()
    })

    it("plays the title's sound as it opens", () => {
        const play = vi.fn()
        render(<BattleTeaser battle={battle} cards={cards} play={play} onClose={vi.fn()}/>)

        expect(play).toHaveBeenCalledWith("title")
    })

    it("asked for, plays no sound and lets the player go at once", async () => {
        const play = vi.fn()
        const onClose = vi.fn()
        render(<BattleTeaser battle={battle} cards={cards} asked play={play} onClose={onClose}/>)

        expect(play).not.toHaveBeenCalled()
        expect(document.querySelector(".title-reveal-flash")).toBeNull()
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "To the globe"}))
        expect(onClose).toHaveBeenCalledOnce()
    })

    it("counts down to the end of a battle that plays", () => {
        render(<BattleTeaser battle={{...battle, startsAt: now - 15 * 60_000, endsAt: now + 45 * 60_000}} cards={cards} asked
                             play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByRole("timer").textContent).toBe("Ends in 45m 00s")
    })
})
