// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RulesPanel} from "./RulesPanel.tsx"

afterEach(cleanup)

const inForceLine = () => screen.findByText((_, element) =>
    element?.tagName === "P" && (element.textContent ?? "").startsWith("In force:"))

const rowOf = (set: string) =>
    screen.getAllByRole("row").find(row => row.querySelector("td")?.textContent === set) as HTMLElement

describe("the game rules page", () => {
    it("says the set in force, and who put it there", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        const line = await inForceLine()
        expect(line.textContent).toContain("In force: rush")
        expect(line.textContent).toContain("by seasons")
        expect(screen.getByText("season-0:finale")).toBeTruthy()
    })

    it("lists every set with its refill and its boxes, and offers the ones not in force", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        await inForceLine()
        const rush = rowOf("rush")
        expect(within(rush).getByText("3× as fast")).toBeTruthy()
        expect(within(rush).getByText("Every 1 min to 2 min, 30 an hour at most")).toBeTruthy()
        expect(within(rush).getByText("A crowd from 50 networks")).toBeTruthy()
        expect(within(rush).getByText("In force")).toBeTruthy()
        expect(screen.queryByRole("button", {name: "Apply rush"})).toBeNull()
        expect(screen.getByRole("button", {name: "Apply default"})).toBeTruthy()
    })

    it("applies a set once confirmed, and reads the rules again", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Apply default"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, apply default"}))

        await waitFor(async () => expect((await inForceLine()).textContent).toContain("In force: default"))
        const [asked] = admin.askedFor("applyRuleSet") as {requestId: string, ruleSet: string}[]
        expect(asked.ruleSet).toBe("default")
        expect(asked.requestId).not.toBe("")
        expect(screen.getByText("An operator")).toBeTruthy()
        expect(screen.getByRole("button", {name: "Apply rush"})).toBeTruthy()
    })

    it("says a refusal in plain words", async () => {
        const admin = new FakeAdmin()
        admin.refuse("applyRuleSet", Code.Unauthenticated)
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Apply default"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, apply default"}))

        expect(await screen.findByText("Your session ended. Reload the page and sign in again.")).toBeTruthy()
    })
})
