// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {NameColor, PlayerTitle} from "../../backends/player.ts"
import {ClosedRound} from "../../backends/standings.ts"
import {BATTLE_BRIEF} from "../../domain/battleBrief.ts"
import BattleBrief from "./BattleBrief.tsx"

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
    takes: [{countryCode: "de", tiles: 4210}, {countryCode: "fr", tiles: 3105}],
    players: [
        {rank: 1, accountId: "account-ana", name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 912, wornTitle: WARLORD},
        {rank: 2, accountId: "account-kofi", name: "Kofi", color: NameColor.GREEN, countryCode: "de", tiles: 640},
    ],
}

const user = () => userEvent.setup({advanceTimers: vi.advanceTimersByTime})

const after = (seconds: number) => act(() => vi.advanceTimersByTime(seconds * 1000))

const renderBrief = (props: Partial<Parameters<typeof BattleBrief>[0]> = {}) =>
    render(<BattleBrief closed={FINAL_BATTLE} description="" countryCode="fr" play={vi.fn()} onClose={vi.fn()} {...props}/>)

describe("BattleBrief", () => {
    it("says the battle is over and who won it, with the battle's times the points", () => {
        renderBrief()

        const dialog = screen.getByRole("dialog", {name: "The Final Battle is over!"})
        expect(within(dialog).getByText("Points ×3")).toBeDefined()
        expect(within(dialog).getByText("Germany")).toBeDefined()
        expect(within(dialog).getByText("+75")).toBeDefined()
        expect(within(dialog).getByLabelText("Rank 2")).toBeDefined()
    })

    it("names a battle that does not end the season as the battle", () => {
        renderBrief({closed: {...FINAL_BATTLE, finale: false}})

        expect(screen.getByRole("dialog", {name: "The battle is over!"})).toBeDefined()
    })

    it("keeps the words of the battle the page saw, and says none it did not", () => {
        renderBrief({description: "Triple points. A bonus box every minute."})
        expect(screen.getByText("Triple points. A bonus box every minute.")).toBeDefined()
        cleanup()

        renderBrief()
        expect(screen.queryByText(/bonus box/)).toBeNull()
    })

    it("plays the fanfare as it opens", () => {
        const play = vi.fn()
        renderBrief({play})

        expect(play).toHaveBeenCalledWith("title")
    })

    it("moves on to the tiles each flag took, the player's marked", () => {
        renderBrief()
        after(BATTLE_BRIEF.podium)

        const takes = screen.getByRole("list", {name: "Tiles taken"})
        expect(within(takes).getAllByRole("listitem").map((row) => row.textContent)).toEqual(["Germany4210", "France3105"])
        expect(within(takes).getByText("France").closest("li")!.className).toContain("battle-brief-row-mine")
    })

    it("ends on the top players: name, flag, tiles and the title worn, and closes once settled", async () => {
        const onClose = vi.fn()
        renderBrief({accountId: "account-kofi", onClose})
        after(BATTLE_BRIEF.podium)
        after(BATTLE_BRIEF.takes)

        const players = screen.getByRole("list", {name: "Top players"})
        const ana = within(players).getByText("Ana").closest("li")!
        expect(within(ana).getByRole("img", {name: "France"})).toBeDefined()
        expect(within(ana).getByRole("img", {name: "Warlord"})).toBeDefined()
        expect(within(ana).getByText("912")).toBeDefined()
        expect(within(players).getByText("Kofi").closest("li")!.getAttribute("aria-current")).toBe("true")

        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()
        after(BATTLE_BRIEF.settle)
        await user().click(screen.getByRole("button", {name: "Close"}))
        expect(onClose).toHaveBeenCalledOnce()
    })

    it("can skip to the next stage, and offers no Next on the last", async () => {
        renderBrief()

        await user().click(screen.getByRole("button", {name: "Next"}))
        expect(screen.getByRole("list", {name: "Tiles taken"})).toBeDefined()
        await user().click(screen.getByRole("button", {name: "Next"}))
        expect(screen.getByRole("list", {name: "Top players"})).toBeDefined()
        expect(screen.queryByRole("button", {name: "Next"})).toBeNull()
    })

    it("skips what the battle has none of", () => {
        renderBrief({closed: {...FINAL_BATTLE, standings: [], players: []}})

        expect(screen.getByRole("list", {name: "Tiles taken"})).toBeDefined()
        expect(screen.queryByRole("button", {name: "Next"})).toBeNull()
    })
})
