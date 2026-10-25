// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook, waitFor} from "@testing-library/react"
import {StrictMode} from "react"
import {LEAVE_GRACE_MS, useTileHolder} from "./useTileHolder.ts"
import {TileHolder} from "../../backends/backend.ts"
import {NameColor, PlayerLine} from "../../backends/player.ts"
import {Peek} from "../../domain/tilePeek.ts"

const ANA: PlayerLine = {
    accountId: "account-ana", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.PINK, streak: 12,
}

const PEEK: Peek = {tile: 7, countryCode: "fr", point: {x: 300, y: 200}, touch: false}

const holdersOf = (holder: TileHolder | undefined) => ({tileHolder: vi.fn(async () => holder)})
const linesOf = (line: PlayerLine | undefined) => ({playerLine: vi.fn(async () => line)})

const ANAS_TILE = {accountId: "account-ana", countryCode: "fr", takenAt: 1_000}

describe("useTileHolder", () => {
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("names the holder of the tile pointed at, beside the pointer, in StrictMode too", async () => {
        const lines = linesOf(ANA)

        const {result} = renderHook(
            () => useTileHolder(holdersOf(ANAS_TILE), lines, {peek: PEEK, id: 1}, "account-me"),
            {wrapper: StrictMode},
        )

        await waitFor(() => expect(result.current.holder).toEqual({line: ANA, takenAt: 1_000, point: PEEK.point, touch: false, id: 1}))
        expect(lines.playerLine).toHaveBeenLastCalledWith("account-ana", "fr", expect.any(AbortSignal))
    })

    it("names nobody on a tile that changed hands since its last take was kept", async () => {
        const holders = holdersOf({...ANAS_TILE, countryCode: "de"})
        const lines = linesOf(ANA)

        const {result} = renderHook(() => useTileHolder(holders, lines, {peek: PEEK, id: 1}, undefined))

        await waitFor(() => expect(holders.tileHolder).toHaveBeenCalled())
        expect(lines.playerLine).not.toHaveBeenCalled()
        expect(result.current.holder).toBeUndefined()
    })

    it("goes a moment after the pointer leaves the tile, unless the pointer is on the card", async () => {
        const holders = holdersOf(ANAS_TILE)
        const lines = linesOf(ANA)
        const {result, rerender} = renderHook(
            ({pointing}) => useTileHolder(holders, lines, pointing, undefined),
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
