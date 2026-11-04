// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {StrictMode} from "react"
import HolderCardPlace from "./HolderCardPlace.tsx"
import {DELETED_ACCOUNT, LEAVE_GRACE_MS} from "./tileHolder.ts"
import {useTileHolder} from "./tileHolderStore.ts"
import {TileHolder} from "../../backends/backend.ts"
import {NameColor, PlayerInfo, PlayerLine} from "../../backends/player.ts"
import {Peek} from "../../domain/tilePeek.ts"

const ANA: PlayerLine = {
    accountId: "account-ana", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.PINK,
}

const PEEK: Peek = {tile: 7, countryCode: "fr", point: {x: 300, y: 200}, touch: false}

const ANA_INFO: PlayerInfo = {
    name: "Ana", guest: false, tilesTaken: 4_000, streakCurrent: 12, streakBest: 30, admin: true, color: NameColor.PINK,
    titles: [], playsFor: [{countryCode: "fr", tiles: 3_000}], playsAgainst: [],
}

const holdersOf = (holder: TileHolder | undefined) => ({tileHolder: vi.fn(async () => holder)})
const playersOf = (info: PlayerInfo | undefined) => ({playerInfo: vi.fn(async () => info)})

const ANAS_TILE = {accountId: "account-ana", countryCode: "fr", takenAt: 1_000}

const POINTING = {peek: PEEK, id: 1}

const shown = () => useTileHolder.getState().holder

describe("HolderCardPlace", () => {
    afterEach(() => {
        cleanup()
        useTileHolder.setState({holder: undefined, heldId: undefined})
        vi.useRealTimers()
    })

    it("reads the holder of the tile pointed at from its public card, and shows it beside the pointer, in StrictMode too", async () => {
        const players = playersOf(ANA_INFO)

        render(<StrictMode>
            <HolderCardPlace holders={holdersOf(ANAS_TILE)} players={players} pointing={POINTING} accountId="account-me"/>
        </StrictMode>)

        await waitFor(() => expect(shown()).toEqual({
            line: ANA, playsFor: ANA_INFO.playsFor, mine: false, takenAt: 1_000, point: PEEK.point, touch: false, id: 1,
        }))
        expect(players.playerInfo).toHaveBeenLastCalledWith("account-ana")
        expect(screen.getByRole("region", {name: "Ana"}).textContent).toContain("Took it for France")
    })

    it("says so when the tile pointed at is the player's own", async () => {
        render(<HolderCardPlace holders={holdersOf(ANAS_TILE)} players={playersOf(ANA_INFO)} pointing={POINTING} accountId="account-ana"/>)

        await waitFor(() => expect(shown()?.mine).toBe(true))
        expect(shown()?.line).toEqual(ANA)
        expect(screen.getByRole("region", {name: "Ana"}).textContent).toContain("You took it for France")
    })

    it("names nobody on a tile that changed hands since its last take was kept", async () => {
        const holders = holdersOf({...ANAS_TILE, countryCode: "de"})
        const players = playersOf(ANA_INFO)

        render(<HolderCardPlace holders={holders} players={players} pointing={POINTING}/>)

        await waitFor(() => expect(holders.tileHolder).toHaveBeenCalled())
        expect(players.playerInfo).not.toHaveBeenCalled()
        expect(shown()).toBeUndefined()
        expect(screen.queryByRole("region")).toBeNull()
    })

    it("shows a take whose account was deleted as a deleted account, and reads no card", async () => {
        const players = playersOf(ANA_INFO)

        render(<HolderCardPlace holders={holdersOf({...ANAS_TILE, accountId: ""})} players={players} pointing={POINTING}/>)

        await waitFor(() => expect(shown()).toEqual({
            line: {accountId: "", name: DELETED_ACCOUNT, countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED},
            playsFor: [], mine: false, takenAt: 1_000, point: PEEK.point, touch: false, id: 1,
        }))
        expect(players.playerInfo).not.toHaveBeenCalled()
    })

    it("goes a moment after the pointer leaves the tile, unless the pointer is on the card", async () => {
        const holders = holdersOf(ANAS_TILE)
        const players = playersOf(ANA_INFO)
        const view = render(<HolderCardPlace holders={holders} players={players} pointing={POINTING}/>)
        await waitFor(() => expect(shown()).toBeDefined())
        vi.useFakeTimers()

        act(() => useTileHolder.getState().hold(true))
        view.rerender(<HolderCardPlace holders={holders} players={players}/>)
        act(() => void vi.advanceTimersByTime(LEAVE_GRACE_MS * 2))
        expect(shown()).toBeDefined()

        act(() => useTileHolder.getState().hold(false))
        act(() => void vi.advanceTimersByTime(LEAVE_GRACE_MS))
        expect(shown()).toBeUndefined()
        expect(screen.queryByRole("region")).toBeNull()
    })

    it("closes from the card's close button", async () => {
        render(<HolderCardPlace holders={holdersOf(ANAS_TILE)} players={playersOf(ANA_INFO)} pointing={POINTING}/>)
        await screen.findByRole("region", {name: "Ana"})

        await userEvent.click(screen.getByRole("button", {name: "Close"}))

        expect(shown()).toBeUndefined()
        expect(screen.queryByRole("region")).toBeNull()
    })

    it("leaves nothing shown once it goes", async () => {
        const view = render(<HolderCardPlace holders={holdersOf(ANAS_TILE)} players={playersOf(ANA_INFO)} pointing={POINTING}/>)
        await waitFor(() => expect(shown()).toBeDefined())

        view.unmount()

        expect(shown()).toBeUndefined()
    })
})
