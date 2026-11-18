// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, waitFor, within} from "@testing-library/react"
import {StrictMode} from "react"
import TerritoryCardPlace, {TerritoryCardPlaceProps} from "./TerritoryCardPlace.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {fortifiedOf, topOf, useTerritoryPlayers} from "./territoryPlayersStore.ts"
import {LandmassState, ProvinceState} from "../../backends/backend.ts"
import {NameColor, PlayerInfo} from "../../backends/player.ts"
import {Territory} from "../../domain/territory.ts"
import {Countries} from "../../domain/countries.ts"
import {useCountry} from "../countryStore.ts"

const infoOf = (name: string): PlayerInfo => ({
    name, guest: false, tilesTaken: 100, streakCurrent: 4, streakBest: 9, admin: false, color: NameColor.TEAL,
    titles: [], playsFor: [], pastNames: [], playsAgainst: [],
})

const KNOWN: Record<string, PlayerInfo> = {
    "account-ana": infoOf("Ana"),
    "account-kiran": infoOf("kiran_07"),
    "account-zoe": infoOf("zoe_nz"),
    "account-mateus": infoOf("Mateus"),
}

const players = () => ({playerInfo: vi.fn(async (accountId: string) => KNOWN[accountId])})

const readerOf = (state: LandmassState) => ({landmass: vi.fn(async () => state)})

const provincesOf = (state: ProvinceState) => ({province: vi.fn(async () => state)})

const SICILY: LandmassState = {
    holders: [
        {accountId: "account-ana", countryCode: "fr", tiles: 40},
        {accountId: "account-ghost", countryCode: "de", tiles: 30},
        {accountId: "account-kiran", countryCode: "in", tiles: 3},
        {accountId: "account-zoe", countryCode: "nz", tiles: 2},
    ],
}

const CALABRIA: ProvinceState = {lastFortification: {accountId: "account-mateus", countryCode: "br", at: 1_000}}

const territoryOf = (landmass: number, province = landmass * 10): Territory => ({
    landmass, name: `Landmass ${landmass}`, ground: "it", size: 100, holdings: [{flag: "fr", tiles: 40}],
    province, provinceName: `Province ${province}`, fortifiedBy: "br", target: undefined,
})

const shownOf = (landmass: number, id = 1, province?: number): TerritoryShown =>
    ({territory: territoryOf(landmass, province), point: {x: 300, y: 200}, touch: false, id})

const readTop = (landmass: number | undefined) => topOf(landmass)(useTerritoryPlayers.getState())

const readFortified = (province: number | undefined) => fortifiedOf(province)(useTerritoryPlayers.getState())

function place(props: Omit<TerritoryCardPlaceProps, "onHold" | "onClose">) {
    useCountry.setState({country: Countries.get("de")!})
    return <TerritoryCardPlace onHold={() => {}} onClose={() => {}} {...props}/>
}

describe("TerritoryCardPlace", () => {
    afterEach(() => {
        cleanup()
        useTerritoryPlayers.getState().forget()
        useCountry.setState(useCountry.getInitialState(), true)
    })

    it("names the first holders of the territory shown and the last fortifier of its province, in StrictMode too", async () => {
        const reader = readerOf(SICILY)
        const provinces = provincesOf(CALABRIA)

        render(<StrictMode>{place({shown: shownOf(12), reader, provinces, players: players()})}</StrictMode>)

        await waitFor(() => expect(readTop(12)).toBeDefined())
        expect(readTop(12)!.map(({line, tiles}) => [line.name, line.countryCode, tiles])).toEqual([
            ["Ana", "fr", 40],
            ["kiran_07", "in", 3],
        ])
        await waitFor(() => expect(readFortified(120)).toBeDefined())
        expect(readFortified(120)!.by).toEqual({line: expect.objectContaining({name: "Mateus", countryCode: "br"}), at: 1_000})
        expect(reader.landmass).toHaveBeenLastCalledWith(12, expect.any(AbortSignal))
        expect(provinces.province).toHaveBeenLastCalledWith(120, expect.any(AbortSignal))

        const top = screen.getByRole("region", {name: "Top players"})
        expect(within(top).getAllByRole("listitem").map((row) => row.querySelector(".territory-card-player-name-text")?.textContent))
            .toEqual(["Ana", "kiran_07"])
        expect(screen.getByRole("region", {name: "Landmass 12"}).textContent).toContain("Mateus")
    })

    it("names a fortifier whose account was deleted as such", async () => {
        const provinces = provincesOf({lastFortification: {accountId: "", countryCode: "fr", at: 1_000}})

        render(place({shown: shownOf(12), reader: readerOf({holders: []}), provinces, players: players()}))

        await waitFor(() => expect(readFortified(120)).toBeDefined())
        expect(readFortified(120)!.by?.line.name).toBe("Deleted account")
    })

    it("reads the last fortify again as the pointer moves to another province of the territory", async () => {
        const reader = readerOf(SICILY)
        const provinces = provincesOf(CALABRIA)
        const backend = players()
        const view = render(place({shown: shownOf(12), reader, provinces, players: backend}))
        await waitFor(() => expect(readFortified(120)).toBeDefined())

        view.rerender(place({shown: shownOf(12, 1, 121), reader, provinces, players: backend}))

        await waitFor(() => expect(readFortified(121)).toBeDefined())
        expect(readFortified(120)).toBeUndefined()
        expect(reader.landmass).toHaveBeenCalledTimes(1)
    })

    it("says nothing about another territory than the one asked", async () => {
        const reader = readerOf(SICILY)
        const provinces = provincesOf(CALABRIA)
        const backend = players()
        const view = render(place({shown: shownOf(12), reader, provinces, players: backend}))
        await screen.findByRole("region", {name: "Top players"})

        view.rerender(place({shown: shownOf(13, 2), reader, provinces, players: backend}))
        expect(screen.queryByRole("region", {name: "Top players"})).toBeNull()
        expect(readTop(13)).toBeUndefined()

        view.rerender(place({reader, provinces, players: backend}))
        expect(screen.queryByRole("region")).toBeNull()
    })

    it("shows the players read already when the same territory opens again, and reads them again", async () => {
        const reader = readerOf(SICILY)
        const provinces = provincesOf(CALABRIA)
        const backend = players()
        const view = render(place({shown: shownOf(12), reader, provinces, players: backend}))
        await screen.findByRole("region", {name: "Top players"})

        view.rerender(place({reader, provinces, players: backend}))
        view.rerender(place({shown: shownOf(12, 2), reader, provinces, players: backend}))

        expect(screen.getByRole("region", {name: "Top players"})).toBeDefined()
        expect(reader.landmass).toHaveBeenCalledTimes(2)
    })

    it("keeps nothing once it goes", async () => {
        const view = render(place({shown: shownOf(12), reader: readerOf(SICILY), provinces: provincesOf(CALABRIA), players: players()}))
        await waitFor(() => expect(readTop(12)).toBeDefined())
        await waitFor(() => expect(readFortified(120)).toBeDefined())

        view.unmount()

        expect(readTop(12)).toBeUndefined()
        expect(readFortified(120)).toBeUndefined()
    })
})
