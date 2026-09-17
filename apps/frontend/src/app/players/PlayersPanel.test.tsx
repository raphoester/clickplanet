// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {RosterEntry} from "../../backends/player.ts"
import {authorHue} from "../../domain/authorColor.ts"
import PlayersPanel from "./PlayersPanel.tsx"

afterEach(cleanup)

const entry = (name: string, guest: boolean, countryCode = "fr", tag = "4f2ca1"): RosterEntry =>
    ({name, tag, countryCode, guest})

const group = (name: string) => screen.queryByRole("region", {name: new RegExp(`^${name}`)})
const names = (region: HTMLElement) =>
    within(region).getAllByRole("listitem").map((li) => li.querySelector(".players-entry-name")!.textContent)

describe("PlayersPanel", () => {
    it("lists players with a username above the guests", () => {
        render(<PlayersPanel entries={[entry("ana", false), entry("guest_Bo", true), entry("kiran_07", false)]}/>)

        expect(names(group("Players")!)).toEqual(["ana", "kiran_07"])
        expect(names(group("Guests")!)).toEqual(["guest_Bo"])

        const [players, guests] = screen.getAllByRole("region")
        expect(players).toBe(group("Players"))
        expect(guests).toBe(group("Guests"))
    })

    it("leaves out a group with nobody in it", () => {
        render(<PlayersPanel entries={[entry("guest_Bo", true)]}/>)

        expect(group("Players")).toBeNull()
        expect(group("Guests")).not.toBeNull()
    })

    it("says so when nobody is playing", () => {
        render(<PlayersPanel entries={[]}/>)

        expect(screen.getByText("Nobody is playing right now.")).toBeDefined()
        expect(screen.queryByRole("list")).toBeNull()
    })

    it("shows each player's flag, named for its country, and tag", () => {
        render(<PlayersPanel entries={[entry("ana", false, "jp", "0c77e2")]}/>)

        const row = screen.getByRole("listitem")
        expect(within(row).getByRole("img", {name: "Japan"}).querySelector(".country-flag")).not.toBeNull()
        expect(within(row).getByText("#0c77e2")).toBeDefined()
    })

    it("cuts a long name as the chat does, keeping the whole of it in the tooltip", () => {
        const long = "guest_" + "x".repeat(24)
        render(<PlayersPanel entries={[entry(long, true)]}/>)

        const name = document.querySelector(".players-entry-name")!
        expect(name.textContent).toBe("guest_xxxxxxxxxx…")
        expect(name.getAttribute("title")).toBe(long)
    })

    // One player is one colour, here and in the chat.
    it("colours a name with the hue the chat gives it", () => {
        render(<PlayersPanel entries={[entry("ana", false, "fr", "4f2ca1")]}/>)

        const row = screen.getByRole("listitem") as HTMLElement
        expect(row.style.getPropertyValue("--author-hue")).toBe(String(authorHue("ana", "4f2ca1")))
    })

    it("opens a player, guest or not, from its name", async () => {
        const onOpenPlayer = vi.fn()
        const bo = entry("guest_Bo", true, "de", "91aa3d")
        render(<PlayersPanel entries={[entry("ana", false), bo]} onOpenPlayer={onOpenPlayer}/>)

        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))

        expect(onOpenPlayer).toHaveBeenCalledWith(bo)
    })

    it("offers no button with nothing to open", () => {
        render(<PlayersPanel entries={[entry("ana", false)]}/>)

        expect(screen.queryByRole("button")).toBeNull()
    })
})
