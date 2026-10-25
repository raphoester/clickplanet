// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {RulesPanel} from "./RulesPanel.tsx"

afterEach(cleanup)

const rowOf = (set: string) =>
    screen.getAllByRole("row").find(row => row.querySelector("td")?.textContent === set) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

describe("the game rules page", () => {
    it("lists every set with its refill, its boxes and its crowd", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        await screen.findByRole("button", {name: "Edit default"})
        const row = rowOf("default")
        expect(within(row).getByText("As configured")).toBeTruthy()
        expect(within(row).getByText("Every 4 min to 8 min, 12 an hour at most")).toBeTruthy()
        expect(within(row).getByText("A crowd from 6 networks")).toBeTruthy()
    })

    it("edits a set once confirmed, and reads the rules again", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        type("Refill, times as fast", "3")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(within(rowOf("default")).getByText("3× as fast")).toBeTruthy())
        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.name).toBe("default")
        expect(asked.set.tempo).toBe(3)
        expect(screen.queryByRole("region", {name: "Edit default"})).toBeNull()
    })

    it("makes a new set", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "New set"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        type("Name", "rush")
        type("Refill, times as fast", "3")
        type("Networks in a crowd", "50")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(rowOf("rush")).toBeTruthy())
        expect(within(rowOf("rush")).getByText("A crowd from 50 networks")).toBeTruthy()
    })

    it("says a refusal in plain words and keeps the form", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveRuleSet", Code.InvalidArgument)
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        expect(await screen.findByRole("alert")).toBeTruthy()
        expect(screen.getByRole("region", {name: "Edit default"})).toBeTruthy()
    })
})
