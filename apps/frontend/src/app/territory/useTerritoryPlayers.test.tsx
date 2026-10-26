// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, renderHook, waitFor} from "@testing-library/react"
import {StrictMode} from "react"
import {useTerritoryPlayers} from "./useTerritoryPlayers.ts"
import {LandmassState} from "../../backends/backend.ts"
import {NameColor, PlayerInfo} from "../../backends/player.ts"
import {DELETED_ACCOUNT} from "../players/useTileHolder.ts"

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

describe("useTerritoryPlayers", () => {
    afterEach(cleanup)

    it("names the first holders and the last fortifier of the territory, in StrictMode too", async () => {
        const reader = readerOf(SICILY)

        const {result} = renderHook(() => useTerritoryPlayers(reader, players(), 12), {wrapper: StrictMode})

        await waitFor(() => expect(result.current).toBeDefined())
        expect(result.current!.top.map(({line, tiles}) => [line.name, line.countryCode, tiles])).toEqual([
            ["Ana", "fr", 40],
            ["kiran_07", "in", 3],
        ])
        expect(result.current!.fortifier).toEqual({line: expect.objectContaining({name: "Mateus", countryCode: "br"}), at: 1_000})
        expect(reader.landmass).toHaveBeenLastCalledWith(12, expect.any(AbortSignal))
    })

    it("names a fortifier whose account was deleted as such", async () => {
        const reader = readerOf({holders: [], lastFortification: {accountId: "", countryCode: "fr", at: 1_000}})

        const {result} = renderHook(() => useTerritoryPlayers(reader, players(), 12))

        await waitFor(() => expect(result.current).toBeDefined())
        expect(result.current!.fortifier?.line.name).toBe(DELETED_ACCOUNT)
    })

    it("says nothing about another territory than the one asked", async () => {
        const reader = readerOf(SICILY)
        const {result, rerender} = renderHook(({landmass}) => useTerritoryPlayers(reader, players(), landmass),
            {initialProps: {landmass: 12 as number | undefined}})
        await waitFor(() => expect(result.current).toBeDefined())

        rerender({landmass: 13})
        expect(result.current).toBeUndefined()

        rerender({landmass: undefined})
        expect(result.current).toBeUndefined()
    })
})
