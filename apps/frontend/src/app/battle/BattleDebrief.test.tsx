// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {NameColor, PlayerTitle} from "../../backends/player.ts"
import {ClosedRound} from "../../backends/standings.ts"
import {BATTLE_DEBRIEF} from "../../domain/battleBrief.ts"
import BattleDebrief from "./BattleDebrief.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

const WARLORD: PlayerTitle = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}

const FINAL_BATTLE: ClosedRound = {
    season: 0,
    number: 1,
    endedAt: Date.UTC(2026, 9, 31, 23),
    finale: true,
    standings: [
        {rank: 1, countryCode: "de", share: 0.3, points: 75},
        {rank: 2, countryCode: "fr", share: 0.2, points: 54},
        {rank: 3, countryCode: "es", share: 0.1, points: 45},
    ],
    before: [],
    after: [],
    own: true,
    times: 3,
    startedAt: Date.UTC(2026, 9, 31, 21),
    entryId: "entry-final",
    takes: [],
    players: [
        {rank: 1, accountId: "account-ana", name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 912, wornTitle: WARLORD},
        {rank: 2, accountId: "account-kofi", name: "Kofi", color: NameColor.GREEN, countryCode: "de", tiles: 640},
    ],
}

const user = () => userEvent.setup({advanceTimers: vi.advanceTimersByTime})

const after = (seconds: number) => act(() => vi.advanceTimersByTime(seconds * 1000))

const renderDebrief = (props: Partial<Parameters<typeof BattleDebrief>[0]> = {}) =>
    render(<BattleDebrief screen="over" closed={FINAL_BATTLE} description="" countryCode="fr" more={true} play={vi.fn()}
                          onClose={vi.fn()} {...props}/>)

describe("BattleDebrief", () => {
    it("says the battle finished, with the fanfare", () => {
        const play = vi.fn()
        renderDebrief({play})

        expect(screen.getByRole("dialog", {name: "Final Battle finished!"})).toBeDefined()
        expect(play).toHaveBeenCalledWith("title")
    })

    it("names a battle that does not end the season as the battle", () => {
        renderDebrief({closed: {...FINAL_BATTLE, finale: false}})

        expect(screen.getByRole("dialog", {name: "Battle finished!"})).toBeDefined()
    })

    it("keeps the words of the battle the page saw, and says none it did not", () => {
        renderDebrief({description: "Triple points. A bonus box every minute."})
        expect(screen.getByText("Triple points. A bonus box every minute.")).toBeDefined()
        cleanup()

        renderDebrief()
        expect(screen.queryByText(/bonus box/)).toBeNull()
    })

    it("goes on once settled, and nothing closes it before", async () => {
        const onClose = vi.fn()
        renderDebrief({onClose})

        expect(screen.queryByRole("button", {name: "Next"})).toBeNull()
        after(BATTLE_DEBRIEF.over)
        await user().click(screen.getByRole("button", {name: "Next"}))
        expect(onClose).toHaveBeenCalledOnce()
    })

    it("breaks the points down: each country, its points times the battle's, the player's marked", () => {
        renderDebrief({screen: "points"})

        const dialog = screen.getByRole("dialog", {name: "Points won"})
        expect(within(dialog).getByText("Points ×3")).toBeDefined()
        const points = screen.getByRole("list", {name: "Points won"})
        expect(within(points).getAllByRole("listitem").map((row) => row.textContent))
            .toEqual(["1Germany25 ×375", "2France18 ×354", "3Spain15 ×345"])
        expect(within(points).getByText("France").closest("li")!.getAttribute("aria-current")).toBe("true")
    })

    it("names the top players: flag, tiles and the title worn, the player found by account", () => {
        renderDebrief({screen: "players", accountId: "account-kofi"})

        const players = screen.getByRole("list", {name: "Top players"})
        const ana = within(players).getByText("Ana").closest("li")!
        expect(within(ana).getByRole("img", {name: "France"})).toBeDefined()
        expect(within(ana).getByRole("img", {name: "Warlord"})).toBeDefined()
        expect(within(ana).getByText("912")).toBeDefined()
        expect(within(players).getByText("Kofi").closest("li")!.getAttribute("aria-current")).toBe("true")
    })

    it("sends the player back to the globe from the last screen", async () => {
        const onClose = vi.fn()
        renderDebrief({screen: "players", more: false, onClose})

        after(BATTLE_DEBRIEF.players)
        await user().click(screen.getByRole("button", {name: "To the globe"}))
        expect(onClose).toHaveBeenCalledOnce()
    })
})
