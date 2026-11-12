// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import TerritoryCard from "./TerritoryCard.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {Territory} from "../../domain/territory.ts"
import {NameColor, PlayerLine} from "../../backends/player.ts"
import {TerritoryPlayers} from "./territoryPlayersStore.ts"
import {deletedLineOf} from "../players/tileHolder.ts"
import {usePlayerCard} from "../players/playerCardStore.ts"
import {useBadges} from "../badges/badgesStore.ts"

const SICILY: Territory = {
    landmass: 12,
    name: "Sicily",
    ground: "it",
    size: 1_200,
    holdings: [
        {flag: "fr", tiles: 600},
        {flag: "it", tiles: 300},
        {flag: "de", tiles: 100},
        {flag: "es", tiles: 50},
        {flag: "pt", tiles: 20},
        {flag: "be", tiles: 10},
        {flag: "nl", tiles: 1},
    ],
    fortifiedBy: "it",
    target: undefined,
}

const NOW = 1_761_393_600_000

const lineOf = (name: string, countryCode: string, guest = false): PlayerLine => ({
    accountId: `account-${name}`, name, countryCode, guest, admin: false, color: NameColor.TEAL,
})

const PLAYERS: TerritoryPlayers = {
    top: [{line: lineOf("Ana", "fr"), tiles: 400}, {line: lineOf("guest_91aa3d", "it", true), tiles: 12}],
    fortifier: {line: lineOf("Mateus", "it"), at: NOW - 3 * 60 * 60_000},
}

// Their streaks are their badges, kept by account.
const STREAKED: TerritoryPlayers = {
    top: [{line: {...lineOf("Ana", "fr"), admin: true}, tiles: 400}],
    fortifier: {line: lineOf("Mateus", "it"), at: NOW - 3 * 60 * 60_000},
}

const STREAKS = {"account-Ana": {streak: 12}, "account-Mateus": {streak: 4}}

const shown = (territory: Territory): TerritoryShown => ({territory, point: {x: 300, y: 200}, touch: false, id: 1})

function show(territory: Territory, props: Partial<Parameters<typeof TerritoryCard>[0]> = {}) {
    render(<TerritoryCard shown={shown(territory)} flag="de" now={NOW} onHold={() => {}} onClose={() => {}} {...props}/>)
}

const rows = () => within(screen.getAllByRole("list")[0]).getAllByRole("listitem").map((row) => row.textContent)

describe("TerritoryCard", () => {
    afterEach(() => {
        cleanup()
        usePlayerCard.setState({player: undefined})
        useBadges.getState().serve(undefined)
    })

    it("names the territory and its size, beside the pointer", () => {
        show(SICILY)

        const card = screen.getByRole("region", {name: "Sicily"})
        expect(screen.getByRole("heading", {name: "Sicily"})).toBeTruthy()
        expect(card.textContent).toContain(`${(1_200).toLocaleString()} tiles, ${(119).toLocaleString()} free`)
        expect(card.style.left).toBe("316px")
        expect(card.style.top).toBe("216px")
    })

    it("ranks the first five flags by the tiles they hold, with their share, and counts the rest", () => {
        show(SICILY)

        expect(rows()).toEqual(["1France60050%", "2Italy30025%", "3Germany1008%", "4Spain504%", "5Portugal202%"])
        expect(screen.getByText("+2 more flags")).toBeTruthy()
    })

    it("marks the player's own flag as on the board", () => {
        show(SICILY)

        const own = screen.getAllByRole("listitem")[2]
        expect(own.querySelector(".coin-you")?.textContent).toBe("3")
    })

    it("says which flag fortified it last", () => {
        show(SICILY)

        expect(screen.getByText(/Fortified by/).textContent).toBe("Fortified by Italy")
    })

    it("says when nobody ever fortified it, and nothing free when no tile is", () => {
        show({...SICILY, size: 1_081, fortifiedBy: undefined})

        expect(screen.getByText("Never fortified")).toBeTruthy()
        expect(screen.getByRole("region", {name: "Sicily"}).textContent).not.toContain("free")
    })

    it("says how many tiles the flag close to fortifying still misses", () => {
        show({...SICILY, target: {flag: "fr", missing: 23}})

        const target = screen.getByText(/to fortify/)
        expect(target.textContent).toContain("23 tiles to fortify")
        expect(within(target).getByRole("img", {name: "France"})).toBeTruthy()
    })

    it("lists no flag on a territory nobody holds", () => {
        show({...SICILY, holdings: [], fortifiedBy: undefined})

        expect(screen.queryByRole("list")).toBeNull()
        expect(screen.getByRole("region", {name: "Sicily"}).textContent).toContain(`${(1_200).toLocaleString()} free`)
    })

    it("lists the players who hold the most of it, and closes to open a signed-in one's card", () => {
        const onClose = vi.fn()
        show(SICILY, {players: PLAYERS, onClose})

        const top = screen.getByRole("region", {name: "Top players"})
        expect(within(top).getAllByRole("listitem").map((row) => row.textContent)).toEqual(["Ana400", "guest_91aa3d12"])
        expect(within(top).queryByRole("button", {name: /guest_91aa3d/})).toBeNull()

        fireEvent.click(within(top).getByRole("button", {name: /Ana/}))
        expect(onClose).toHaveBeenCalledTimes(1)
        expect(usePlayerCard.getState().player).toEqual(PLAYERS.top[0].line)
    })

    it("says a streak when its flame is pressed, and opens no card", () => {
        const onClose = vi.fn()
        useBadges.setState({badges: STREAKS})
        show(SICILY, {players: STREAKED, onClose})

        fireEvent.click(screen.getByRole("button", {name: "12-day streak"}))
        fireEvent.click(screen.getByRole("button", {name: "4-day streak"}))

        expect(screen.getAllByRole("status").map((bubble) => bubble.textContent)).toEqual(["Played 12 days in a row", "Played 4 days in a row"])
        expect(onClose).not.toHaveBeenCalled()
        expect(usePlayerCard.getState().player).toBeUndefined()
    })

    it("opens a card from a name that wears a crown and a flame", () => {
        const onClose = vi.fn()
        useBadges.setState({badges: STREAKS})
        show(SICILY, {players: STREAKED, onClose})

        fireEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(onClose).toHaveBeenCalledTimes(1)
        expect(usePlayerCard.getState().player).toEqual(STREAKED.top[0].line)

        fireEvent.click(screen.getByRole("button", {name: "Mateus"}))
        expect(usePlayerCard.getState().player).toEqual(STREAKED.fortifier?.line)
    })

    it("puts no button inside another", () => {
        const warned = vi.spyOn(console, "error").mockImplementation(() => {})
        useBadges.setState({badges: STREAKS})
        show(SICILY, {players: STREAKED})

        const buttons = within(screen.getByRole("region", {name: "Sicily"})).getAllByRole("button")
        expect(buttons.map((button) => button.textContent)).toEqual(["", "Ana", "12", "Mateus", "4"])
        expect(buttons.filter((button) => button.parentElement?.closest("button"))).toEqual([])
        expect(warned).not.toHaveBeenCalled()
        warned.mockRestore()
    })

    it("lists an account once for each flag it holds the territory for", () => {
        const warned = vi.spyOn(console, "error").mockImplementation(() => {})
        show(SICILY, {players: {top: [{line: lineOf("Ana", "fr"), tiles: 400}, {line: lineOf("Ana", "it"), tiles: 30}], fortifier: undefined}})

        const top = screen.getByRole("region", {name: "Top players"})
        expect(within(top).getAllByRole("listitem").map((row) => row.textContent)).toEqual(["Ana400", "Ana30"])
        expect(warned).not.toHaveBeenCalled()
        warned.mockRestore()
    })

    it("lists no player until they are read", () => {
        show(SICILY)

        expect(screen.queryByRole("region", {name: "Top players"})).toBeNull()
    })

    it("says who fortified it last and when, under the flag that did", () => {
        show(SICILY, {players: PLAYERS})

        expect(screen.getByText("Mateus").closest("p")?.textContent).toBe("Mateus3 hr. ago")
    })

    it("names nobody under a lock a fortify did not set", () => {
        show({...SICILY, fortifiedBy: "fr"}, {players: PLAYERS})

        expect(screen.queryByText("Mateus")).toBeNull()
    })

    it("says a deleted account fortified it", () => {
        show(SICILY, {players: {top: [], fortifier: {line: deletedLineOf("it"), at: NOW}}})

        expect(screen.getByText("Deleted account")).toBeTruthy()
    })

    it("closes from its button", () => {
        const onClose = vi.fn()
        show(SICILY, {onClose})

        fireEvent.click(screen.getByRole("button", {name: "Close"}))

        expect(onClose).toHaveBeenCalled()
    })

    it("names the missions on its ground or on a rival holding some of it, the flag's included", () => {
        show(SICILY, {missions: {endsAt: 0, missions: [
            {id: "ground", order: {kind: "takeIn", country: "it", goal: 300, taken: 0}, reward: {kind: "bomb", amount: 1}, done: false, paid: false},
            {id: "far", order: {kind: "takeIn", country: "br", goal: 300, taken: 0}, reward: {kind: "bomb", amount: 1}, done: false, paid: false},
        ], flagMission: {id: "flag", order: {kind: "breakShields", country: "es", goal: 30, reached: 0}, points: 5, done: false}}})

        const lines = within(screen.getByRole("list", {name: "Your missions here"})).getAllByRole("listitem")
        expect(lines.map((line) => line.textContent?.replace(/\s+/g, " "))).toEqual(["Mission: Take 300 tiles in Italy", "Flag mission: Break 30 shields of Spain"])
    })
})
