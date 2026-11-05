// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor, within} from "@testing-library/react"
import {StrictMode} from "react"
import TerritoryCardPlace, {TerritoryCardPlaceProps} from "./TerritoryCardPlace.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {playersOf, useTerritoryPlayers} from "./territoryPlayersStore.ts"
import {LandmassState} from "../../backends/backend.ts"
import {NameColor, PlayerInfo} from "../../backends/player.ts"
import {Territory} from "../../domain/territory.ts"
import {DELETED_ACCOUNT} from "../players/tileHolder.ts"

const infoOf = (name: string): PlayerInfo => ({
    name, guest: false, tilesTaken: 100, streakCurrent: 4, streakBest: 9, admin: false, color: NameColor.TEAL,
    titles: [], playsFor: [], playsAgainst: [],
})

const KNOWN: Record<string, PlayerInfo> = {
    "account-ana": infoOf("Ana"),
    "account-kiran": infoOf("kiran_07"),
    "account-zoe": infoOf("zoe_nz"),
    "account-mateus": infoOf("Mateus"),
}

const players = () => ({playerInfo: vi.fn(async (accountId: string) => KNOWN[accountId])})

const readerOf = (state: LandmassState) => ({landmass: vi.fn(async () => state)})

const SICILY: LandmassState = {
    holders: [
        {accountId: "account-ana", countryCode: "fr", tiles: 40},
        {accountId: "account-ghost", countryCode: "de", tiles: 30},
        {accountId: "account-kiran", countryCode: "in", tiles: 3},
        {accountId: "account-zoe", countryCode: "nz", tiles: 2},
    ],
    lastFortification: {accountId: "account-mateus", countryCode: "br", at: 1_000},
}

const territoryOf = (landmass: number): Territory => ({
    landmass, name: `Landmass ${landmass}`, ground: "it", size: 100, holdings: [{flag: "fr", tiles: 40}], fortifiedBy: "br", target: undefined,
})

const shownOf = (landmass: number, id = 1): TerritoryShown => ({territory: territoryOf(landmass), point: {x: 300, y: 200}, touch: false, id})

const read = (landmass: number | undefined) => playersOf(landmass)(useTerritoryPlayers.getState())

function place(props: Partial<TerritoryCardPlaceProps>) {
    return <TerritoryCardPlace flag="de" onHold={() => {}} onClose={() => {}} {...props}/>
}

describe("TerritoryCardPlace", () => {
    afterEach(() => {
        cleanup()
        useTerritoryPlayers.setState({landmass: undefined, players: undefined})
    })

    it("names the first holders and the last fortifier of the territory shown, in StrictMode too", async () => {
        const reader = readerOf(SICILY)

        render(<StrictMode>{place({shown: shownOf(12), reader, players: players()})}</StrictMode>)

        await waitFor(() => expect(read(12)).toBeDefined())
        expect(read(12)!.top.map(({line, tiles}) => [line.name, line.countryCode, tiles])).toEqual([
            ["Ana", "fr", 40],
            ["kiran_07", "in", 3],
        ])
        expect(read(12)!.fortifier).toEqual({line: expect.objectContaining({name: "Mateus", countryCode: "br"}), at: 1_000})
        expect(reader.landmass).toHaveBeenLastCalledWith(12, expect.any(AbortSignal))

        const top = screen.getByRole("region", {name: "Top players"})
        expect(within(top).getAllByRole("listitem").map((row) => row.querySelector(".territory-card-player-name-text")?.textContent))
            .toEqual(["Ana", "kiran_07"])
        expect(screen.getByRole("region", {name: "Landmass 12"}).textContent).toContain("Mateus")
    })

    it("names a fortifier whose account was deleted as such", async () => {
        const reader = readerOf({holders: [], lastFortification: {accountId: "", countryCode: "fr", at: 1_000}})

        render(place({shown: shownOf(12), reader, players: players()}))

        await waitFor(() => expect(read(12)).toBeDefined())
        expect(read(12)!.fortifier?.line.name).toBe(DELETED_ACCOUNT)
    })

    it("says nothing about another territory than the one asked", async () => {
        const reader = readerOf(SICILY)
        const backend = players()
        const view = render(place({shown: shownOf(12), reader, players: backend}))
        await screen.findByRole("region", {name: "Top players"})

        view.rerender(place({shown: shownOf(13, 2), reader, players: backend}))
        expect(screen.queryByRole("region", {name: "Top players"})).toBeNull()
        expect(read(13)).toBeUndefined()

        view.rerender(place({reader, players: backend}))
        expect(screen.queryByRole("region")).toBeNull()
    })

    it("shows the players read already when the same territory opens again, and reads them again", async () => {
        const reader = readerOf(SICILY)
        const backend = players()
        const view = render(place({shown: shownOf(12), reader, players: backend}))
        await screen.findByRole("region", {name: "Top players"})

        view.rerender(place({reader, players: backend}))
        view.rerender(place({shown: shownOf(12, 2), reader, players: backend}))

        expect(screen.getByRole("region", {name: "Top players"})).toBeDefined()
        expect(reader.landmass).toHaveBeenCalledTimes(2)
    })

    it("keeps nothing once it goes", async () => {
        const view = render(place({shown: shownOf(12), reader: readerOf(SICILY), players: players()}))
        await waitFor(() => expect(read(12)).toBeDefined())

        view.unmount()

        expect(read(12)).toBeUndefined()
    })
})
