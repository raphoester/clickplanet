// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {OpsPage} from "./OpsPage.tsx"

afterEach(cleanup)

function confirm(label: string, yes: string) {
    fireEvent.click(screen.getByRole("button", {name: label}))
    fireEvent.click(screen.getByRole("button", {name: yes}))
}

describe("OpsPage", () => {
    it("reconciles the titles once confirmed, and says how many moved", async () => {
        const admin = new FakeAdmin()
        render(<OpsPage clients={admin.clients()}/>)

        confirm("Reconcile titles", "Yes, reconcile titles")

        expect(await screen.findByText("Granted 12 titles, revoked 3.")).toBeTruthy()
        expect(admin.askedFor("reconcileTitles")).toEqual([{}])
    })

    it("names the accounts once confirmed", async () => {
        const admin = new FakeAdmin()
        render(<OpsPage clients={admin.clients()}/>)

        confirm("Name accounts", "Yes, name accounts")

        expect(await screen.findByText("Named 7 accounts.")).toBeTruthy()
    })

    it("says a session that ended", async () => {
        const admin = new FakeAdmin()
        admin.refuse("nameAccounts", Code.Unauthenticated)
        render(<OpsPage clients={admin.clients()}/>)

        confirm("Name accounts", "Yes, name accounts")

        expect(await screen.findByText("Your session ended. Reload the page and sign in again.")).toBeTruthy()
    })
})
