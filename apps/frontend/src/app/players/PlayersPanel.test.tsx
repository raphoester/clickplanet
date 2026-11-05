// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {NameColor, RosterEntry} from "../../backends/player.ts"
import PlayersPanel from "./PlayersPanel.tsx"
import {usePlayerCard} from "./playerCardStore.ts"
import {useRoster} from "./rosterStore.ts"
import {useBadges} from "../badges/badgesStore.ts"

afterEach(() => {
    cleanup()
    usePlayerCard.setState({openable: false, player: undefined})
    useRoster.setState({state: {kind: "unavailable"}})
    useBadges.getState().serve(undefined)
})

const entry = (name: string, guest: boolean, countryCode = "fr", admin = false, color = NameColor.UNSPECIFIED): RosterEntry =>
    ({key: name, accountId: `account-${name}`, name, countryCode, guest, admin, color})

function show(entries: RosterEntry[]) {
    useRoster.setState({state: {kind: "ready", entries}})
    return render(<PlayersPanel/>)
}

const group = (name: string) => screen.queryByRole("region", {name: new RegExp(`^${name}`)})
const names = (region: HTMLElement) =>
    within(region).getAllByRole("listitem").map((li) => li.querySelector(".players-entry-name")!.textContent)

describe("PlayersPanel", () => {
    it("lists players with a username above the guests", () => {
        show([entry("ana", false), entry("guest_Bo", true), entry("kiran_07", false)])

        expect(names(group("Players")!)).toEqual(["ana", "kiran_07"])
        expect(names(group("Guests")!)).toEqual(["guest_Bo"])

        const [players, guests] = screen.getAllByRole("region")
        expect(players).toBe(group("Players"))
        expect(guests).toBe(group("Guests"))
    })

    it("leaves out a group with nobody in it", () => {
        show([entry("guest_Bo", true)])

        expect(group("Players")).toBeNull()
        expect(group("Guests")).not.toBeNull()
    })

    it("says so when nobody is playing", () => {
        show([])

        expect(screen.getByText("Nobody is playing right now.")).toBeDefined()
        expect(screen.queryByRole("list")).toBeNull()
    })

    it("shows each player's flag, named for its country, and nothing of its address", () => {
        show([entry("ana", false, "jp")])

        const row = screen.getByRole("listitem")
        expect(within(row).getByRole("img", {name: "Japan"}).querySelector(".country-flag")).not.toBeNull()
        expect(row.textContent).toBe("ana")
    })

    it("shows a guest's name once", () => {
        show([entry("guest_a1b2c3", true)])

        expect(screen.getByRole("listitem").textContent).toBe("guest_a1b2c3")
    })

    it("cuts a long name as the chat does, keeping the whole of it in the tooltip", () => {
        const long = "guest_" + "x".repeat(24)
        show([entry(long, true)])

        const name = document.querySelector(".players-entry-name")!
        expect(name.textContent).toBe("guest_xxxxxxxxxx…")
        expect(name.getAttribute("title")).toBe(long)
    })

    it("colours a name in the color its player chose, and grey with none", () => {
        show([entry("ana", false, "fr", false, NameColor.TEAL), entry("bo", false)])

        const [ana, bo] = screen.getAllByRole("listitem") as HTMLElement[]
        expect(ana.style.getPropertyValue("--author-hue")).toBe("165")
        expect(ana.style.getPropertyValue("--author-chroma")).toBe("")
        expect(bo.style.getPropertyValue("--author-chroma")).toBe("0")
    })

    it("opens a player, guest or not, from its name", async () => {
        usePlayerCard.setState({openable: true})
        const bo = entry("guest_Bo", true, "de")
        show([entry("ana", false), bo])

        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))

        expect(usePlayerCard.getState().player).toEqual(bo)
    })

    it("offers no button with nothing to open", () => {
        show([entry("ana", false)])

        expect(screen.queryByRole("button")).toBeNull()
    })

    it("shows the badges kept for each line's account beside its name, and nothing beside one that has none", () => {
        useBadges.setState({badges: {"account-ana": {streak: 12, wornTitle: {id: "og", name: "OG"}}, "account-kiran_07": {streak: 0}}})
        show([entry("ana", false), entry("kiran_07", false)])

        const [ana, kiran] = screen.getAllByRole("listitem")
        expect(within(ana).getByRole("img", {name: "OG"}).querySelector(".title-emblem")).not.toBeNull()
        expect(within(ana).getByRole("img", {name: "12-day streak"})).toBeDefined()
        expect(within(kiran).queryByRole("img", {name: "OG"})).toBeNull()
        expect(within(kiran).queryByRole("img", {name: /streak/})).toBeNull()
    })

    it("crowns an admin, and nobody else", () => {
        show([entry("ana", false, "fr", true), entry("kiran_07", false)])

        const [ana, kiran] = screen.getAllByRole("listitem")
        expect(within(ana).getByRole("img", {name: "Admin"})).toBeDefined()
        expect(within(kiran).queryByRole("img", {name: "Admin"})).toBeNull()
    })
})
