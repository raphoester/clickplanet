// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
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

    it("recounts the bombs and fortifies once confirmed, and says how many players moved", async () => {
        const admin = new FakeAdmin()
        render(<TitlesPanel player={admin}/>)

        confirm("Recount bombs and fortifies", "Yes, recount bombs and fortifies")

        expect(await screen.findByText("Recounted 164 players.")).toBeTruthy()
        expect(admin.askedFor("recountStats")).toEqual([{}])
    })

    it("says a session that ended", async () => {
        const admin = new FakeAdmin()
        admin.refuse("reconcileTitles", Code.Unauthenticated)
        render(<TitlesPanel player={admin}/>)

        confirm("Reconcile titles", "Yes, reconcile titles")

        expect(await screen.findByText("Your session ended. Reload the page and sign in again.")).toBeTruthy()
    })
})
