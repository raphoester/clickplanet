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

const save = () => {
    fireEvent.click(screen.getByRole("button", {name: "Save"}))
    fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))
}

const sent = (admin: FakeAdmin) => (admin.askedFor("saveRuleSet") as {set: RuleSet}[]).at(-1)?.set

describe("the game rules page", () => {
    it("lists every set with its refill, its clicks and its boxes", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        await screen.findByRole("button", {name: "Edit default"})
        const row = rowOf("default")
        expect(within(row).getByText("1×")).toBeTruthy()
        expect(within(row).getByText("60 in hand, 0.2 a second")).toBeTruthy()
        expect(within(row).getByText("Every 4 min to 8 min, 12 an hour at most")).toBeTruthy()
    })

    it("edits a set once confirmed, and reads the rules again", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        expect((screen.getByLabelText("Shields on a tile, at most") as HTMLInputElement).value).toBe("10")
        type("Refill, times as fast", "3")
        type("Shields on a tile, at most", "20")
        type("Time to answer (s)", "6")
        save()

        await waitFor(() => expect(within(rowOf("default")).getByText("3×")).toBeTruthy())
        const set = sent(admin)
        expect(set?.name).toBe("default")
        expect(set?.tempo).toBe(3)
        expect(set?.bonuses?.shield?.perTile).toBe(20)
        expect(Number(set?.quiz?.answerWindow?.seconds)).toBe(6)
        expect(set?.clicks?.burst).toBe(60)
        expect(screen.queryByRole("region", {name: "Edit default"})).toBeNull()
    })

    it("switches quizzes off and edits the toll", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        fireEvent.click(screen.getByLabelText("Quizzes on"))
        fireEvent.click(screen.getByRole("button", {name: "Remove step 1"}))
        fireEvent.click(screen.getByRole("button", {name: "Add a step"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        const added = within(screen.getByRole("group", {name: "Step 3"}))
        fireEvent.change(added.getByLabelText("From % of the map"), {target: {value: "40"}})
        fireEvent.change(added.getByLabelText("Times slower"), {target: {value: "6"}})
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.quiz?.enabled).toBe(false)
        expect(sent(admin)?.tollSteps.map(step => [step.share, step.slowdown])).toEqual([[0.2, 2.5], [0.3, 4], [0.4, 6]])
    })

    it("makes a new set from the one played", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "New set"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).value).toBe("")
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        type("Name", "rush")
        type("Refill, times as fast", "3")
        save()

        await waitFor(() => expect(rowOf("rush")).toBeTruthy())
        expect(sent(admin)?.bonuses?.kinds?.refill).toBe(5)
    })

    it("quotes the server's refusal and keeps the form", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveRuleSet", Code.InvalidArgument)
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        save()

        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
        expect(screen.getByRole("region", {name: "Edit default"})).toBeTruthy()
    })
})
