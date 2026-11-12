// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import {FakePromoBackend} from "../../backends/fakePromoBackend.ts"
import {useAnnouncements} from "../announcements/announcementsStore.ts"
import BattleChip, {BattleDetails} from "./BattleChip.tsx"
import {battleEndsLine} from "./battleLines.ts"
import {useScreen} from "../hud/screenStore.ts"
import {NO_SCHEDULE, useSchedule} from "./scheduleStore.ts"
import {useTeaserCards} from "./teaserCardsStore.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE

const now = Date.UTC(2026, 9, 31, 21, 30)
const battle = {entryId: "entry-final", since: now - 30 * MINUTE, endsAt: now + HOUR + 23 * MINUTE + 5 * SECOND, description: "Triple points.", roundId: "entry-final"}
const season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt: Date.UTC(2026, 11, 5, 23), bookId: "season-0"}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    useScreen.setState(useScreen.getInitialState(), true)
    useSchedule.setState(NO_SCHEDULE)
    useAnnouncements.setState({queue: []})
    useTeaserCards.getState().serve(undefined)
})

describe("BattleChip", () => {
    it("counts the battle down, to the second", () => {
        vi.useFakeTimers({now})
        render(<BattleChip battle={battle} compact={false}/>)

        expect(screen.getByRole("region", {name: "Battle"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Battle ends in 1h 23m 05s"})).toBeDefined()

        act(() => {
            vi.advanceTimersByTime(5 * SECOND)
        })
        expect(screen.getByRole("button", {name: "Battle ends in 1h 23m 00s"})).toBeDefined()
    })

    it("names the season's finale the final battle", () => {
        vi.useFakeTimers({now})
        const finale = {...battle, entryId: "", roundId: "season-0/finale", endsSeason: true}
        render(<BattleChip battle={finale} compact={false}/>)

        expect(screen.getByRole("region", {name: "Final Battle"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Final Battle ends in 1h 23m 05s"})).toBeDefined()
        cleanup()

        render(<BattleChip battle={finale} compact={true}/>)
        expect(screen.getByRole("button", {name: "Final Battle ends in 1h 23m 05s"}).textContent).toContain("Final")
    })

    it("tells the battle's facts on hover, and on a press", () => {
        vi.useFakeTimers({now})
        render(<BattleChip battle={battle} season={season} compact={false}/>)
        expect(screen.queryByText("Triple points.")).toBeNull()

        fireEvent.pointerEnter(screen.getByRole("region", {name: "Battle"}))
        expect(screen.getByText("Triple points.")).toBeDefined()
        expect(screen.getByText(battleEndsLine(battle, now))).toBeDefined()
        expect(screen.getByText("Season ends in")).toBeDefined()

        fireEvent.pointerLeave(screen.getByRole("region", {name: "Battle"}))
        expect(screen.queryByText("Triple points.")).toBeNull()

        fireEvent.click(screen.getByRole("button", {name: /Battle ends in/}))
        expect(screen.getByText("Triple points.")).toBeDefined()
        expect(useScreen.getState().popovers.battle).toBe(true)
    })

    it("holds its facts open once pressed", () => {
        vi.useFakeTimers({now})
        useScreen.setState({popovers: {season: false, battle: true, clicks: false}})
        render(<BattleChip battle={battle} compact={false}/>)

        expect(screen.getByText("Triple points.")).toBeDefined()
        expect(screen.getByRole("button", {name: /Battle ends in/}).getAttribute("aria-expanded")).toBe("true")
    })

    it("is a tagged pill on a phone, the two largest units alone", () => {
        vi.useFakeTimers({now})
        render(<BattleChip battle={battle} compact/>)

        const chip = screen.getByRole("button", {name: "Battle ends in 1h 23m 05s"})
        expect(chip.textContent).toBe("Battle1h 23m")
    })

    it("has a sheet with the clock and the facts", () => {
        vi.useFakeTimers({now})
        render(<BattleDetails battle={battle}/>)

        expect(screen.getByRole("timer", {name: "Battle ends in"}).textContent).toBe("1h 23m 05s")
        expect(screen.getByText("Triple points.")).toBeDefined()
    })

    it("opens the rules of the battle that plays, counted down to its end", async () => {
        vi.useFakeTimers({now})
        const promo = new FakePromoBackend()
        useTeaserCards.getState().serve(promo)
        useSchedule.getState().hearTurn({kind: "battle", since: battle.since, endsAt: battle.endsAt, description: battle.description,
            entryId: battle.entryId, roundId: battle.roundId, times: 3, endsSeason: false})
        render(<BattleDetails battle={battle}/>)
        await act(async () => {
            await Promise.resolve()
        })

        fireEvent.click(screen.getByRole("button", {name: "See the rules"}))

        expect(promo.asked).toEqual(["entry-final"])
        expect(useAnnouncements.getState().queue[0]).toMatchObject({
            kind: "teaser", asked: true, battle: {entryId: "entry-final", startsAt: battle.since, endsAt: battle.endsAt, times: 3},
        })
    })
})
