// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {ALERTS_OFF} from "../players/actions.ts"
import {DiscordMembersPanel} from "./DiscordMembersPanel.tsx"
import {membersLinkedWords} from "./discordMembers.ts"
import {TitlesPanel} from "./TitlesPanel.tsx"

afterEach(cleanup)

function confirm(label: string, yes: string) {
    fireEvent.click(screen.getByRole("button", {name: label}))
    fireEvent.click(screen.getByRole("button", {name: yes}))
}

describe("the ops runs", () => {
    it("reconciles the titles once confirmed, and says how many moved", async () => {
        const admin = new FakeAdmin()
        render(<TitlesPanel player={admin}/>)

        confirm("Reconcile titles", "Yes, reconcile titles")

        expect(await screen.findByText("Granted 12 titles, revoked 3.")).toBeTruthy()
        expect(admin.askedFor("reconcileTitles")).toEqual([{}])
    })

    it("says a session that ended", async () => {
        const admin = new FakeAdmin()
        admin.refuse("reconcileTitles", Code.Unauthenticated)
        render(<TitlesPanel player={admin}/>)

        confirm("Reconcile titles", "Yes, reconcile titles")

        expect(await screen.findByText("Your session ended. Reload the page and sign in again.")).toBeTruthy()
    })
})

describe("linking the Discord members", () => {
    it("links once confirmed, and says who is in the server and who is not", async () => {
        const admin = new FakeAdmin()
        render(<DiscordMembersPanel alerts={admin.alerts}/>)

        confirm("Link Discord members", "Yes, link discord members")

        expect(await screen.findByText(
            "Linked 7 players. 13 players not in the server: the game offers them \"Join Discord\".")).toBeTruthy()
        expect(admin.askedFor("linkMembers")).toEqual([{}])
    })

    it("says Discord pings are off when the server does not serve them", async () => {
        const admin = new FakeAdmin()
        admin.refuse("linkMembers", Code.Unimplemented)
        render(<DiscordMembersPanel alerts={admin.alerts}/>)

        confirm("Link Discord members", "Yes, link discord members")

        expect(await screen.findByText(ALERTS_OFF)).toBeTruthy()
    })
})

describe("what a run of the Discord members says", () => {
    it("says there was nothing to do", () => {
        expect(membersLinkedWords({linked: 0, outside: 0, failed: 0}))
            .toBe("Nothing to do: every player signed in with Discord is linked already.")
    })

    it("says nobody new was in the server, and who could not be checked", () => {
        expect(membersLinkedWords({linked: 0, outside: 1, failed: 2})).toBe("Nobody new was in the server. "
            + "1 player not in the server: the game offers them \"Join Discord\". "
            + "2 players could not be checked: run it again, and the server's log says why.")
    })
})
