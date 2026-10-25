// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, renderHook, waitFor} from "@testing-library/react"
import {StrictMode} from "react"
import {useTileHolder} from "./useTileHolder.ts"
import {TileHolder} from "../../backends/backend.ts"
import {NameColor, PlayerLine} from "../../backends/player.ts"

const ANA: PlayerLine = {
    accountId: "account-ana", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.PINK, streak: 12,
}

const holdersOf = (holder: TileHolder | undefined) => ({tileHolder: vi.fn(async () => holder)})
const linesOf = (line: PlayerLine | undefined) => ({playerLine: vi.fn(async () => line)})

describe("useTileHolder", () => {
    afterEach(cleanup)

    it("names the holder of the tile peeked at, in StrictMode too", async () => {
        const holders = holdersOf({accountId: "account-ana", countryCode: "fr", takenAt: 1_000})
        const lines = linesOf(ANA)

        const {result} = renderHook(
            () => useTileHolder(holders, lines, {peek: {tile: 7, countryCode: "fr"}, id: 1}, "account-me"),
            {wrapper: StrictMode},
        )

        await waitFor(() => expect(result.current.holder?.line).toEqual(ANA))
        expect(result.current.holder?.takenAt).toBe(1_000)
        expect(lines.playerLine).toHaveBeenLastCalledWith("account-ana", "fr", expect.any(AbortSignal))
    })

    it("names nobody on a tile that changed hands since its last take was kept", async () => {
        const holders = holdersOf({accountId: "account-ana", countryCode: "de", takenAt: 1_000})
        const lines = linesOf(ANA)

        const {result} = renderHook(() => useTileHolder(holders, lines, {peek: {tile: 7, countryCode: "fr"}, id: 1}, undefined))

        await waitFor(() => expect(holders.tileHolder).toHaveBeenCalled())
        expect(lines.playerLine).not.toHaveBeenCalled()
        expect(result.current.holder).toBeUndefined()
    })
})
