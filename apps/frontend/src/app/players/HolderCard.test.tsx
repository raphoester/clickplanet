// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import HolderCard from "./HolderCard.tsx"
import {usePlayerCard} from "./playerCardStore.ts"
import {DELETED_ACCOUNT} from "./tileHolder.ts"
import {HolderShown} from "./tileHolderStore.ts"
import {NameColor, PlayerLine} from "../../backends/player.ts"
import {useBadges} from "../badges/badgesStore.ts"

const NOW = 1_761_393_600_000

const FOX: PlayerLine = {
    accountId: "account-fox", name: "BraveFox42", countryCode: "fr", guest: false, admin: false,
    color: NameColor.TEAL,
}

const WARLORD = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}

const GUEST: PlayerLine = {
    accountId: "account-guest", name: "guest_91aa3d", countryCode: "de", guest: true, admin: false,
    color: NameColor.UNSPECIFIED,
}

const pressed = (pointerType: string) => Object.assign(new Event("pointerdown", {bubbles: true}), {pointerType})

const FRONTS = [{countryCode: "fr", tiles: 9_000}, {countryCode: "be", tiles: 300}]

const shown = (line: PlayerLine): HolderShown => ({
    line, tile: {owner: line.countryCode, ground: "dz", shields: 0}, playsFor: line.guest ? [] : FRONTS, mine: false, takenAt: NOW - 3 * 60_000, point: {x: 300, y: 200}, touch: false, id: 1,
})

function show(line: PlayerLine, props: Partial<Parameters<typeof HolderCard>[0]> = {}) {
    render(<HolderCard holder={shown(line)} now={NOW} onHold={() => {}} onClose={() => {}} {...props}/>)
}

describe("HolderCard", () => {
    beforeEach(() => {
        useBadges.setState({badges: {[FOX.accountId]: {streak: 4, wornTitle: WARLORD}}})
    })

    afterEach(() => {
        cleanup()
        usePlayerCard.setState({player: undefined})
        useBadges.getState().serve(undefined)
    })

    it("says who holds the tile, the title it wears and when it took it, beside the pointer", () => {
        show(FOX)

        const card = screen.getByRole("region", {name: "BraveFox42"})
        expect(card.textContent).toContain("BraveFox42")
        expect(screen.getByRole("img", {name: "Warlord"})).toBeTruthy()
        expect(card.textContent).not.toContain("Warlord")
        expect(screen.getByText(/Took it for France 3 min ago/)).toBeTruthy()
        expect(card.style.left).toBe("316px")
        expect(card.style.top).toBe("216px")
    })

    it("says the player took the tile itself when it is its own", () => {
        show(FOX, {holder: {...shown(FOX), mine: true}})

        expect(screen.getByText(/You took it for France 3 min ago/)).toBeTruthy()
    })

    it("shows the first flags the player fought for", () => {
        show(FOX)

        const flags = screen.getByRole("group", {name: "Fought for"})
        expect(within(flags).getAllByRole("img").map((flag) => flag.getAttribute("aria-label"))).toEqual(["France", "Belgium"])
    })

    it("closes, then opens the player's card, from its name", () => {
        const onClose = vi.fn()
        show(FOX, {onClose})

        fireEvent.click(screen.getByRole("button", {name: "BraveFox42"}))

        expect(onClose).toHaveBeenCalledTimes(1)
        expect(usePlayerCard.getState().player).toEqual(FOX)
    })

    it("shows a guest by its guest name alone, with no profile to open", () => {
        show(GUEST)

        expect(screen.getByRole("region", {name: "guest_91aa3d"}).textContent).toContain("Took it for Germany 3 min ago")
        expect(screen.queryByRole("button", {name: "guest_91aa3d"})).toBeNull()
        expect(screen.queryByRole("group", {name: "Fought for"})).toBeNull()
    })

    it("shows a deleted account by that alone, with no profile to open", () => {
        const deleted: PlayerLine = {...GUEST, accountId: "", name: DELETED_ACCOUNT, countryCode: "tr", guest: false}
        render(<HolderCard holder={{...shown(deleted), playsFor: []}} now={NOW} onHold={() => {}} onClose={() => {}}/>)

        expect(screen.getByRole("region", {name: "Deleted account"}).textContent).toContain("Took it for Turkey 3 min ago")
        expect(screen.queryByRole("button", {name: "Deleted account"})).toBeNull()
    })

    it("says when the pointer is on it, and closes on Escape", () => {
        const onHold = vi.fn()
        const onClose = vi.fn()
        show(FOX, {onHold, onClose})
        const card = screen.getByRole("region", {name: "BraveFox42"})

        fireEvent.pointerEnter(card)
        fireEvent.pointerLeave(card)
        fireEvent.keyDown(document, {key: "Escape"})

        expect(onHold.mock.calls).toEqual([[true], [false]])
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("closes from its close button, and not on a press inside it", () => {
        const onClose = vi.fn()
        show(FOX, {onClose})

        document.querySelector(".holder-card-name")!.dispatchEvent(pressed("touch"))
        expect(onClose).not.toHaveBeenCalled()

        fireEvent.click(screen.getByRole("button", {name: "Close"}))
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("closes on a tap anywhere else, and that tap does not reach what is under it", () => {
        const onClose = vi.fn()
        const globe = vi.fn()
        const outside = document.body.appendChild(document.createElement("div"))
        outside.addEventListener("click", globe)
        show(FOX, {onClose})

        outside.dispatchEvent(pressed("touch"))
        fireEvent.click(outside)

        expect(onClose).toHaveBeenCalledTimes(1)
        expect(globe).not.toHaveBeenCalled()
        outside.remove()
    })

    it("closes on a mouse press elsewhere and lets that click through, for a click that paints", () => {
        const onClose = vi.fn()
        const globe = vi.fn()
        const outside = document.body.appendChild(document.createElement("div"))
        outside.addEventListener("click", globe)
        show(FOX, {onClose})

        outside.dispatchEvent(pressed("mouse"))
        fireEvent.click(outside)

        expect(onClose).toHaveBeenCalledTimes(1)
        expect(globe).toHaveBeenCalledTimes(1)
        outside.remove()
    })

    it("names the missions the map draws on the tile, after a ring of their red", () => {
        show(FOX, {missions: {endsAt: 0, missions: [
            {id: "rival", order: {kind: "takeFrom", country: "fr", goal: 80, taken: 3}, reward: {kind: "bomb", amount: 1}, done: false, paid: false},
            {id: "ground", order: {kind: "takeIn", country: "br", goal: 300, taken: 0}, reward: {kind: "bomb", amount: 1}, done: false, paid: false},
        ]}})

        const here = screen.getByRole("list", {name: "Your missions here"})
        expect(here.textContent?.replace(/\s+/g, " ")).toBe("Mission: Take 80 tiles from France")
    })

    it("names no mission on a tile the map draws nothing on", () => {
        show(FOX, {missions: {endsAt: 0, missions: [
            {id: "ground", order: {kind: "takeIn", country: "br", goal: 300, taken: 0}, reward: {kind: "bomb", amount: 1}, done: false, paid: false},
        ]}})

        expect(screen.queryByRole("list", {name: "Your missions here"})).toBeNull()
    })
})
