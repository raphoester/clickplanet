// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import TileHolderNews from "./TileHolderNews.tsx"
import {NameColor, PlayerLine} from "../../backends/player.ts"

const NOW = 1_761_393_600_000

const FOX: PlayerLine = {
    accountId: "account-fox", name: "BraveFox42", countryCode: "fr", guest: false, admin: false,
    color: NameColor.TEAL, streak: 4,
    wornTitle: {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}},
}

const ended = (animationName: string) => Object.assign(new Event("animationend", {bubbles: true}), {animationName})

const GUEST: PlayerLine = {
    accountId: "account-guest", name: "guest_91aa3d", countryCode: "de", guest: true, admin: false,
    color: NameColor.UNSPECIFIED, streak: 0,
}

describe("TileHolderNews", () => {
    afterEach(cleanup)

    it("names who holds the tile, the title it wears and how long ago it took it", () => {
        render(<TileHolderNews line={FOX} takenAt={NOW - 3 * 60_000} now={NOW} onDone={() => {}}/>)

        expect(screen.getByRole("status").textContent).toBe("BraveFox42Warlord· 3 min ago")
    })

    it("names a guest by its guest name alone", () => {
        render(<TileHolderNews line={GUEST} takenAt={NOW - 20_000} now={NOW} onDone={() => {}}/>)

        expect(screen.getByRole("status").textContent).toBe("guest_91aa3d· just now")
    })

    it("opens the card of the account on the line", () => {
        const onOpenPlayer = vi.fn()
        render(<TileHolderNews line={FOX} takenAt={NOW} now={NOW} onOpenPlayer={onOpenPlayer} onDone={() => {}}/>)

        fireEvent.click(screen.getByRole("button", {name: "BraveFox42"}))

        expect(onOpenPlayer).toHaveBeenCalledWith(FOX)
    })

    it("goes once it has faded out, and not when it pops in", () => {
        const onDone = vi.fn()
        render(<TileHolderNews line={FOX} takenAt={NOW} now={NOW} onDone={onDone}/>)
        const line = screen.getByRole("status").firstElementChild!

        fireEvent(line, ended("tile-holder-news-pop"))
        expect(onDone).not.toHaveBeenCalled()

        fireEvent(line, ended("tile-holder-news"))
        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
