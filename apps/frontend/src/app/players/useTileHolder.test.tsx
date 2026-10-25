// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook, waitFor} from "@testing-library/react"
import {StrictMode} from "react"
import {DELETED_ACCOUNT, LEAVE_GRACE_MS, useTileHolder} from "./useTileHolder.ts"
import {TileHolder} from "../../backends/backend.ts"
import {NameColor, PlayerInfo, PlayerLine} from "../../backends/player.ts"
import {Peek} from "../../domain/tilePeek.ts"

const ANA: PlayerLine = {
    accountId: "account-ana", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.PINK, streak: 12,
}

const PEEK: Peek = {tile: 7, countryCode: "fr", point: {x: 300, y: 200}, touch: false}

const ANA_INFO: PlayerInfo = {
    name: "Ana", guest: false, tilesTaken: 4_000, streakCurrent: 12, streakBest: 30, admin: true, color: NameColor.PINK,
    titles: [], playsFor: [{countryCode: "fr", tiles: 3_000}], playsAgainst: [],
}

const holdersOf = (holder: TileHolder | undefined) => ({tileHolder: vi.fn(async () => holder)})
const playersOf = (info: PlayerInfo | undefined) => ({playerInfo: vi.fn(async () => info)})

const ANAS_TILE = {accountId: "account-ana", countryCode: "fr", takenAt: 1_000}

describe("useTileHolder", () => {
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("reads the holder of the tile pointed at from its public card, beside the pointer, in StrictMode too", async () => {
        const players = playersOf(ANA_INFO)

        const {result} = renderHook(
            () => useTileHolder(holdersOf(ANAS_TILE), players, {peek: PEEK, id: 1}, "account-me"),
            {wrapper: StrictMode},
        )

        await waitFor(() => expect(result.current.holder).toEqual({
            line: ANA, playsFor: ANA_INFO.playsFor, takenAt: 1_000, point: PEEK.point, touch: false, id: 1,
        }))
        expect(players.playerInfo).toHaveBeenLastCalledWith("account-ana")
    })

    it("names nobody on a tile that changed hands since its last take was kept", async () => {
        const holders = holdersOf({...ANAS_TILE, countryCode: "de"})
        const players = playersOf(ANA_INFO)

        const {result} = renderHook(() => useTileHolder(holders, players, {peek: PEEK, id: 1}, undefined))

        await waitFor(() => expect(holders.tileHolder).toHaveBeenCalled())
        expect(players.playerInfo).not.toHaveBeenCalled()
        expect(result.current.holder).toBeUndefined()
    })

    it("shows a take whose account was deleted as a deleted account, and reads no card", async () => {
        const players = playersOf(ANA_INFO)

        const {result} = renderHook(() => useTileHolder(holdersOf({...ANAS_TILE, accountId: ""}), players, {peek: PEEK, id: 1}, undefined))

        await waitFor(() => expect(result.current.holder).toEqual({
            line: {accountId: "", name: DELETED_ACCOUNT, countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
            playsFor: [], takenAt: 1_000, point: PEEK.point, touch: false, id: 1,
        }))
        expect(players.playerInfo).not.toHaveBeenCalled()
    })

    it("goes a moment after the pointer leaves the tile, unless the pointer is on the card", async () => {
        const holders = holdersOf(ANAS_TILE)
        const players = playersOf(ANA_INFO)
        const {result, rerender} = renderHook(
            ({pointing}) => useTileHolder(holders, players, pointing, undefined),
            {initialProps: {pointing: {peek: PEEK, id: 1} as {peek: Peek, id: number} | undefined}},
        )
        await waitFor(() => expect(result.current.holder).toBeDefined())
        vi.useFakeTimers()

        act(() => result.current.hold(true))
        rerender({pointing: undefined})
        act(() => void vi.advanceTimersByTime(LEAVE_GRACE_MS * 2))
        expect(result.current.holder).toBeDefined()

        act(() => result.current.hold(false))
        act(() => void vi.advanceTimersByTime(LEAVE_GRACE_MS))
        expect(result.current.holder).toBeUndefined()
    })
})
