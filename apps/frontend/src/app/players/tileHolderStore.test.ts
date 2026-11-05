import {afterEach, describe, expect, it} from "vitest"
import {NameColor} from "../../backends/player.ts"
import {HolderShown, holderHeld, useTileHolder} from "./tileHolderStore.ts"

const shown = (id: number): HolderShown => ({
    line: {accountId: "account-ana", name: "Ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED},
    tile: {owner: "fr", ground: "fr", shields: 0}, playsFor: [], mine: false, takenAt: 1_000, point: {x: 300, y: 200}, touch: false, id,
})

afterEach(() => useTileHolder.setState({holder: undefined, heldId: undefined}))

describe("the tile holder", () => {
    it("is held only while the pointer is on the card shown", () => {
        const card = useTileHolder.getState()
        card.show(shown(1))

        card.hold(true)
        expect(holderHeld(useTileHolder.getState())).toBe(true)

        card.show(shown(2))
        expect(holderHeld(useTileHolder.getState())).toBe(false)

        card.hold(true)
        card.hold(false)
        expect(holderHeld(useTileHolder.getState())).toBe(false)
    })

    it("shows nothing once dismissed", () => {
        const card = useTileHolder.getState()
        card.show(shown(1))
        card.hold(true)

        card.dismiss()

        expect(useTileHolder.getState().holder).toBeUndefined()
        expect(holderHeld(useTileHolder.getState())).toBe(false)
    })
})
