// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {NameColor, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import PlayerCardPlace from "./PlayerCardPlace.tsx"
import {usePlayerCard} from "./playerCardStore.ts"

afterEach(() => {
    cleanup()
    usePlayerCard.setState({openable: false, player: undefined})
})

const ana: PlayerLine = {accountId: "account-ana", name: "Ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED}

const missing: PlayerInfoBackend = {playerInfo: async () => undefined}

describe("PlayerCardPlace", () => {
    it("lets a name open a card only while it is there, and shows the card opened until it is closed", async () => {
        const {unmount} = render(<PlayerCardPlace backend={missing}/>)
        expect(usePlayerCard.getState().openable).toBe(true)

        act(() => usePlayerCard.getState().open(ana))
        expect(screen.getByRole("dialog", {name: "Ana"})).toBeDefined()

        await userEvent.click(screen.getByRole("button", {name: "Close"}))
        expect(screen.queryByRole("dialog")).toBeNull()

        unmount()
        expect(usePlayerCard.getState().openable).toBe(false)
    })
})
