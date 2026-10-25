// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import HolderCard from "./HolderCard.tsx"
import {HolderShown} from "./useTileHolder.ts"
import {NameColor, PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"

const NOW = 1_761_393_600_000

const FOX: PlayerLine = {
    accountId: "account-fox", name: "BraveFox42", countryCode: "fr", guest: false, admin: false,
    color: NameColor.TEAL, streak: 4,
    wornTitle: {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}},
}

const GUEST: PlayerLine = {
    accountId: "account-guest", name: "guest_91aa3d", countryCode: "de", guest: true, admin: false,
    color: NameColor.UNSPECIFIED, streak: 0,
}

const FOX_INFO: PlayerInfo = {
    name: "BraveFox42", tilesTaken: 12_345, streakCurrent: 4, streakBest: 30, createdAt: Date.UTC(2026, 8, 17),
    admin: false, color: NameColor.TEAL, titles: [], wornTitle: FOX.wornTitle,
    playsFor: [{countryCode: "fr", tiles: 9_000}, {countryCode: "be", tiles: 300}], playsAgainst: [],
}

const pressed = (pointerType: string) => Object.assign(new Event("pointerdown", {bubbles: true}), {pointerType})

const shown = (line: PlayerLine): HolderShown => ({line, takenAt: NOW - 3 * 60_000, point: {x: 300, y: 200}, touch: false, id: 1})

const backendOf = (info: PlayerInfo | undefined) => ({playerInfo: vi.fn(async () => info)}) satisfies PlayerInfoBackend

function show(line: PlayerLine, props: Partial<Parameters<typeof HolderCard>[0]> = {}) {
    const backend = backendOf(FOX_INFO)
    render(<HolderCard holder={shown(line)} backend={backend} now={NOW} onHold={() => {}} onClose={() => {}} {...props}/>)
    return backend
}

describe("HolderCard", () => {
    afterEach(cleanup)

    it("says who holds the tile, the title it wears and when it took it, beside the pointer", () => {
        show(FOX)

        const card = screen.getByRole("region", {name: "BraveFox42"})
        expect(card.textContent).toContain("BraveFox42")
        expect(card.textContent).toContain("Warlord")
        expect(screen.getByText(/Took it 3 min ago/)).toBeTruthy()
        expect(card.style.left).toBe("316px")
        expect(card.style.top).toBe("216px")
    })

    it("then says what the player did, read from its public card", async () => {
        const backend = show(FOX)

        expect(await screen.findByText("12,345")).toBeTruthy()
        expect(screen.getByText("30 days")).toBeTruthy()
        expect(screen.getByRole("img", {name: "Belgium"})).toBeTruthy()
        expect(backend.playerInfo).toHaveBeenCalledWith("account-fox")
    })

    it("opens the player's card from its profile button", () => {
        const onOpenPlayer = vi.fn()
        show(FOX, {onOpenPlayer})

        fireEvent.click(screen.getByRole("button", {name: "View profile"}))

        expect(onOpenPlayer).toHaveBeenCalledWith(FOX)
    })

    it("shows a guest by its guest name alone, and asks nothing more about it", () => {
        const backend = show(GUEST, {onOpenPlayer: vi.fn()})

        expect(screen.getByRole("region", {name: "guest_91aa3d"}).textContent).toContain("Took it 3 min ago")
        expect(screen.queryByRole("button", {name: "View profile"})).toBeNull()
        expect(backend.playerInfo).not.toHaveBeenCalled()
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
})
