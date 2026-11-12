// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import {FakePromoBackend} from "../../backends/fakePromoBackend.ts"
import {Finale} from "../../domain/battle.ts"
import {useAnnouncements} from "../announcements/announcementsStore.ts"
import {useTeaserCards} from "../battle/teaserCardsStore.ts"
import SeasonChip, {ChampionChip, SeasonDetails} from "./SeasonChip.tsx"
import {TURN_MS} from "./useRotation.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import {useScreen} from "../hud/screenStore.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt, bookId: "season-0"}
const lastDayEndsAt = endsAt - 2 * HOUR
// Scoring's day in progress ends at the next 21:00 UTC.
const dayEndsAt = () => lastDayEndsAt - Math.floor((lastDayEndsAt - Date.now() - 1) / DAY) * DAY
const battleTeased = () => screen.queryByText(/Final Battle/)

const shownFace = () => [...document.querySelectorAll(".season-chip-face:not(.season-chip-face--away) > *")]
    .map((part) => part.textContent)
const turn = () => act(() => vi.advanceTimersByTime(TURN_MS))

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
    useScreen.setState(useScreen.getInitialState(), true)
    useAnnouncements.setState({queue: []})
    useTeaserCards.getState().serve(undefined)
})

// The rules a press away: the button shows once promo answered the battle's cards.
async function rules() {
    await act(async () => {
        await Promise.resolve()
    })
    return screen.queryByRole("button", {name: "See the rules"})
}

describe("SeasonChip", () => {
    it("counts down to today's cutoff, at the time of day the last day ends", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(shownFace()).toEqual(["Today ends in", "12h 00m 00s"])
        expect(battleTeased()).toBeNull()
    })

    it("turns to the next battle's countdown between today's and the season's", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        const nextBattle = {entryId: "rush", startsAt: Date.now() + DAY, endsAt: Date.now() + DAY + 2 * HOUR, description: "", times: 3, endsSeason: false, roundId: "rush"}
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} nextBattle={nextBattle} compact={false}/>)

        turn()
        expect(shownFace()).toEqual(["Next battle in", "23h 59m 54s"])
        turn()
        expect(shownFace()[0]).toBe("Season ends in")
    })

    it("ticks down every second", () => {
        vi.useFakeTimers({now: lastDayEndsAt - 3 * DAY - (2 * HOUR + 52 * MINUTE + 10 * SECOND)})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)
        expect(shownFace()).toEqual(["Today ends in", "2h 52m 10s"])

        act(() => vi.advanceTimersByTime(SECOND))
        expect(shownFace()).toEqual(["Today ends in", "2h 52m 09s"])
    })

    it("turns to the season's countdown, then back to today's", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)

        act(() => vi.advanceTimersByTime(TURN_MS - SECOND))
        expect(shownFace()).toEqual(["Today ends in", "11h 59m 55s"])

        act(() => vi.advanceTimersByTime(SECOND))
        expect(shownFace()).toEqual(["Season ends in", "27d 13h 59m 54s"])

        turn()
        expect(shownFace()).toEqual(["Today ends in", "11h 59m 48s"])
    })

    it("names both countdowns, and no live region, so a screen reader needs no turn", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        const {container} = render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)

        const name = "Today ends in 12h 00m 00s and Season ends in 27d 14h 00m 00s"
        expect(screen.getByRole("button", {name})).toBeDefined()
        turn()
        expect(screen.getByRole("button", {name: "Today ends in 11h 59m 54s and Season ends in 27d 13h 59m 54s"})).toBeDefined()

        expect(screen.queryByRole("timer")).toBeNull()
        expect(container.querySelector("[aria-live]")).toBeNull()
    })

    it("holds still while the mouse is over it", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)

        fireEvent.mouseEnter(screen.getByRole("region", {name: "Season 0"}))
        turn()
        turn()
        expect(shownFace()[0]).toBe("Today ends in")

        fireEvent.mouseLeave(screen.getByRole("region", {name: "Season 0"}))
        turn()
        expect(shownFace()[0]).toBe("Season ends in")
    })

    it("holds still while it has focus", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)
        const chip = screen.getByRole("button", {name: /Today ends in/})

        act(() => chip.focus())
        turn()
        expect(shownFace()[0]).toBe("Today ends in")

        act(() => chip.blur())
        turn()
        expect(shownFace()[0]).toBe("Season ends in")
    })

    it("counts the season alone once its last day ended, and does not turn", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Season ends in 1h 12m 00s"})).toBeDefined()
        expect(shownFace()).toEqual(["Season ends in", "1h 12m 00s"])

        turn()
        turn()
        expect(shownFace()).toEqual(["Season ends in", "1h 11m 48s"])
    })

    it("opens on what a season is, and promises no battle", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)

        fireEvent.click(screen.getByRole("button", {name: /Today ends in/}))
        expect(screen.getByRole("button", {name: /Today ends in/}).getAttribute("aria-expanded")).toBe("true")
        expect(screen.getByText("When a day ends, the countries that hold the most ground and gained the most that day score points.")).toBeDefined()
        expect(battleTeased()).toBeNull()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(useScreen.getState().popovers.season).toBe(false)
    })

    describe("before a battle", () => {
        const nextBattle = () => ({
            entryId: "rush", startsAt: Date.now() + DAY, endsAt: Date.now() + DAY + 2 * HOUR,
            description: "Clicks refill 3× faster.", times: 3, endsSeason: false, roundId: "rush",
        })
        const opened = () => {
            useScreen.setState({popovers: {season: true, battle: false, clicks: false}})
            render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} nextBattle={nextBattle()} compact={false}/>)
        }

        it("opens on the next battle: when, what it is worth, and the operator's words", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
            opened()

            expect(screen.getByText("Next battle")).toBeDefined()
            expect(screen.getByText(/^Tomorrow · \d\d?:\d\d.+ · Points ×3$/)).toBeDefined()
            expect(screen.getByText("Clicks refill 3× faster.")).toBeDefined()
            expect(battleTeased()).toBeNull()
        })

        it("opens its rules a press away: its teaser, asked for", async () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
            const promo = new FakePromoBackend()
            useTeaserCards.getState().serve(promo)
            opened()

            fireEvent.click((await rules())!)

            expect(promo.asked).toEqual(["rush"])
            expect(useAnnouncements.getState().queue).toEqual([{kind: "teaser", battle: nextBattle(), cards: promo.cards, asked: true}])
        })

        it("has no rules to open while promo has no card for it", async () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
            const promo = new FakePromoBackend()
            promo.cards = []
            useTeaserCards.getState().serve(promo)
            opened()

            expect(await rules()).toBeNull()
            expect(promo.asked).toEqual(["rush"])
        })
    })

    describe("before the Final Battle", () => {
        const startsAt = endsAt - 2 * HOUR
        const finale: Finale = {roundId: "season-0/finale", startsAt, endsAt, times: 3, description: "A bonus box every minute.", live: false}
        const nextBattle = {entryId: "", startsAt, endsAt, description: finale.description, times: 3, endsSeason: true, roundId: finale.roundId}
        const chip = () => <SeasonChip season={season} dayEndsAt={dayEndsAt()} nextBattle={nextBattle} finale={finale}
                                       compact={false}/>

        it("wears the battle's look in the two days before it, and paints its face", () => {
            vi.useFakeTimers({now: startsAt - 3 * DAY})
            const {container} = render(chip())
            expect(container.querySelector(".season-chip--final")).toBeNull()

            vi.setSystemTime(startsAt - 2 * DAY - SECOND)
            act(() => vi.advanceTimersByTime(SECOND))
            expect(container.querySelector(".season-chip--final")).not.toBeNull()
            expect(container.querySelector(".season-chip-face--final")?.textContent).toBe("Final Battle in2d 00h 00m 00s")
        })

        it("opens on the Final Battle: when, how long, what it is worth, and the operator's words", () => {
            vi.useFakeTimers({now: startsAt - DAY})
            useScreen.setState({popovers: {season: true, battle: false, clicks: false}})
            render(chip())

            expect(screen.getByText("The Final Battle")).toBeDefined()
            expect(screen.getByText("A bonus box every minute.")).toBeDefined()
            expect(screen.getByText("2 hours of war")).toBeDefined()
            expect(screen.getByText("Points ×3")).toBeDefined()
            expect(screen.getByText("The countries that gain the most ground during it score the points of 3 days.")).toBeDefined()
            expect(screen.getByText("The last word")).toBeDefined()
            expect(screen.queryByText("Next battle")).toBeNull()
        })

        it("opens its rules once it is the next battle, by its round", async () => {
            vi.useFakeTimers({now: startsAt - DAY})
            const promo = new FakePromoBackend()
            useTeaserCards.getState().serve(promo)
            useScreen.setState({popovers: {season: true, battle: false, clicks: false}})
            render(chip())

            fireEvent.click((await rules())!)

            expect(promo.asked).toEqual(["season-0/finale"])
            expect(useAnnouncements.getState().queue[0]).toMatchObject({kind: "teaser", battle: nextBattle, asked: true})
        })
    })

    it("is gone once the season is over", () => {
        vi.useFakeTimers({now: endsAt})
        const {container} = render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact={false}/>)
        expect(container.innerHTML).toBe("")
    })

    it("tells the moments at the top where it ends, so they slide under it", () => {
        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 16, 376, 51.4))
        const bottom = () => document.documentElement.style.getPropertyValue(STATUS_BOTTOM)

        const {unmount} = render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} dayEndsAt={dayEndsAt()} compact={false}/>)
        expect(bottom()).toBe("68px")

        unmount()
        expect(bottom()).toBe("")
    })

    describe("on a phone", () => {
        it("is a chip with a tag and the two largest units, named in full", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR - 5 * MINUTE})
            render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact/>)

            const chip = screen.getByRole("button", {name: "Today ends in 12h 05m 00s and Season ends in 27d 14h 05m 00s"})
            expect(chip.getAttribute("aria-expanded")).toBe("false")
            expect(shownFace()).toEqual(["Today", "12h 05m"])

            turn()
            expect(shownFace()).toEqual(["Season", "27d 14h"])
        })

        it("holds still while the mouse is over it", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
            render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact/>)

            fireEvent.mouseEnter(screen.getByRole("button"))
            turn()
            expect(shownFace()[0]).toBe("Today")
        })

        it("has no tag with one countdown", () => {
            vi.useFakeTimers({now: endsAt - HOUR})
            render(<SeasonChip season={season} dayEndsAt={dayEndsAt()} compact/>)

            expect(screen.getByRole("button", {name: "Season ends in 1h 00m 00s"})).toBeDefined()
            expect(shownFace()).toEqual(["1h 00m"])
        })

        it("leaves the moments to the status bar", () => {
            render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} dayEndsAt={dayEndsAt()} compact/>)
            expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
        })
    })
})

describe("SeasonDetails", () => {
    it("shows both countdowns at once, and promises no battle", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        render(<SeasonDetails season={season} dayEndsAt={dayEndsAt()}/>)

        expect(screen.getByRole("timer", {name: "Today ends in"}).textContent).toBe("22h 00m 00s")
        expect(screen.getByRole("timer", {name: "Season ends in"}).textContent).toBe("27d 00h 00m 00s")
        expect(battleTeased()).toBeNull()

        turn()
        expect(screen.getByRole("timer", {name: "Today ends in"}).textContent).toBe("21h 59m 54s")
        expect(screen.getByRole("timer", {name: "Season ends in"}).textContent).toBe("26d 23h 59m 54s")
    })

    it("tells the Final Battle to come, and nothing of it once it plays", () => {
        vi.useFakeTimers({now: endsAt - DAY})
        const finale: Finale = {roundId: "season-0/finale", startsAt: endsAt - 2 * HOUR, endsAt, times: 3, description: "", live: false}
        const {rerender} = render(<SeasonDetails season={season} dayEndsAt={dayEndsAt()} finale={finale}/>)
        expect(screen.getByText("The Final Battle")).toBeDefined()
        expect(screen.getByText("Points ×3")).toBeDefined()

        rerender(<SeasonDetails season={season} dayEndsAt={dayEndsAt()} finale={{...finale, live: true}}/>)
        expect(screen.queryByText("The Final Battle")).toBeNull()
    })

    it("tells the next battle, and its rules a press away", async () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        useTeaserCards.getState().serve(new FakePromoBackend())
        const nextBattle = {entryId: "rush", startsAt: Date.now() + 3 * HOUR, endsAt: Date.now() + 4 * HOUR, description: "", times: 1, endsSeason: false, roundId: "rush"}
        render(<SeasonDetails season={season} dayEndsAt={dayEndsAt()} nextBattle={nextBattle}/>)

        expect(screen.getByRole("timer", {name: "Next battle in"}).textContent).toBe("3h 00m 00s")
        expect(screen.getByText("Next battle")).toBeDefined()
        expect(screen.queryByText(/Points ×/)).toBeNull()
        expect(await rules()).not.toBeNull()
    })

    it("counts the season alone once its last day ended", () => {
        vi.useFakeTimers({now: endsAt - HOUR})
        render(<SeasonDetails season={season} dayEndsAt={dayEndsAt()}/>)

        expect(screen.getAllByRole("timer")).toHaveLength(1)
        expect(screen.getByRole("timer", {name: "Season ends in"}).textContent).toBe("1h 00m 00s")
    })
})

describe("ChampionChip", () => {
    const germany = {season: 0, countryCodes: ["de"]}

    it("shows the champion of the season that ended", () => {
        render(<ChampionChip champion={germany} compact={false}/>)

        const chip = screen.getByRole("region", {name: "Germany wins Season 0"})
        expect(chip.textContent).toContain("Season 0 champion")
        expect(chip.textContent).toContain("Germany")
        expect(chip.textContent).toContain("🏆")
    })

    it("names champions together", () => {
        render(<ChampionChip champion={{season: 0, countryCodes: ["de", "fr"]}} compact/>)

        expect(screen.getByRole("region", {name: "Germany and France win Season 0"})).toBeDefined()
    })

    it("tells the moments at the top where it ends, as the countdown did", () => {
        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 16, 376, 51.4))
        const bottom = () => document.documentElement.style.getPropertyValue(STATUS_BOTTOM)

        const {unmount} = render(<ChampionChip champion={germany} compact={false}/>)
        expect(bottom()).toBe("68px")

        unmount()
        expect(bottom()).toBe("")
    })

    it("on a phone, is the trophy and the flag, and leaves the moments to the status bar", () => {
        render(<ChampionChip champion={germany} compact/>)

        const chip = screen.getByRole("region", {name: "Germany wins Season 0"})
        expect(chip.textContent).toBe("🏆")
        expect(chip.querySelectorAll(".country-flag")).toHaveLength(1)
        expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
    })
})
