// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import TerritoryCard from "./TerritoryCard.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {Territory} from "../../domain/territory.ts"
import {NameColor, PlayerLine} from "../../backends/player.ts"
import {TerritoryPlayers} from "./useTerritoryPlayers.ts"
import {deletedLineOf} from "../players/useTileHolder.ts"

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
    accountId: `account-${name}`, name, countryCode, guest, admin: false, color: NameColor.TEAL, streak: 0,
})

const PLAYERS: TerritoryPlayers = {
    top: [{line: lineOf("Ana", "fr"), tiles: 400}, {line: lineOf("guest_91aa3d", "it", true), tiles: 12}],
    fortifier: {line: lineOf("Mateus", "it"), at: NOW - 3 * 60 * 60_000},
}

const shown = (territory: Territory): TerritoryShown => ({territory, point: {x: 300, y: 200}, touch: false, id: 1})

function show(territory: Territory, props: Partial<Parameters<typeof TerritoryCard>[0]> = {}) {
    render(<TerritoryCard shown={shown(territory)} flag="de" now={NOW} onHold={() => {}} onClose={() => {}} {...props}/>)
}

const rows = () => within(screen.getAllByRole("list")[0]).getAllByRole("listitem").map((row) => row.textContent)

describe("TerritoryCard", () => {
    afterEach(cleanup)

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

    it("lists the players who hold the most of it, and opens a signed-in one's card", () => {
        const onOpenPlayer = vi.fn()
        show(SICILY, {players: PLAYERS, onOpenPlayer})

        const top = screen.getByRole("region", {name: "Top players"})
        expect(within(top).getAllByRole("listitem").map((row) => row.textContent)).toEqual(["Ana400", "guest_91aa3d12"])
        expect(within(top).queryByRole("button", {name: /guest_91aa3d/})).toBeNull()

        fireEvent.click(within(top).getByRole("button", {name: /Ana/}))
        expect(onOpenPlayer).toHaveBeenCalledWith(PLAYERS.top[0].line)
    })

    it("lists no player until they are read", () => {
        show(SICILY)

        expect(screen.queryByRole("region", {name: "Top players"})).toBeNull()
    })

    it("says who fortified it last and when, under the flag that did", () => {
        show(SICILY, {players: PLAYERS})

        expect(screen.getByText("Mateus").closest("p")?.textContent).toBe("Mateus3 hours ago")
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
})
