// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
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

// A second set, kept beside default and not playing.
function withCalm(): FakeAdmin {
    const admin = new FakeAdmin()
    const calm = new RuleSet({name: "calm", rules: admin.rules.sets[0].rules?.clone()})
    admin.rules = new GetRulesResponse({sets: [admin.rules.sets[0], calm], playing: "default"})
    return admin
}

describe("the game rules page", () => {
    it("lists every set with its refill, its clicks and its boxes, and says which plays", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.getByText("Playing:").textContent).toContain("default")
        const row = rowOf("default")
        expect(row.getAttribute("aria-current")).toBe("true")
        expect(within(row).getByText("1×")).toBeTruthy()
        expect(within(row).getByText("60 in hand, 0.2 a second")).toBeTruthy()
        expect(within(row).getByText("Every 4 min to 8 min, 12 an hour at most")).toBeTruthy()
    })

    it("offers no edit of the set that plays", async () => {
        render(<RulesPanel planet={withCalm()}/>)

        await screen.findByRole("button", {name: "Edit calm"})
        expect(screen.queryByRole("button", {name: "Edit default"})).toBeNull()
        expect(screen.getByRole("button", {name: "Copy default"})).toBeTruthy()
    })

    it("edits a set once confirmed, and reads the rules again", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        expect((screen.getByLabelText("Shields on a tile, at most") as HTMLInputElement).value).toBe("10")
        type("Refill, times as fast", "3")
        type("Shields on a tile, at most", "20")
        type("Time to answer (s)", "6")
        save()

        await waitFor(() => expect(within(rowOf("calm")).getByText("3×")).toBeTruthy())
        const rules = sent(admin)?.rules
        expect(sent(admin)?.name).toBe("calm")
        expect(rules?.clicks?.tempo).toBe(3)
        expect(rules?.bonuses?.shield?.perTile).toBe(20)
        expect(Number(rules?.bonuses?.quiz?.answerWindow?.seconds)).toBe(6)
        expect(rules?.clicks?.burst).toBe(60)
        expect(screen.queryByRole("region", {name: "Edit calm"})).toBeNull()
    })

    it("switches quizzes off and edits the toll", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        fireEvent.click(screen.getByLabelText("Quizzes on"))
        fireEvent.click(screen.getByRole("button", {name: "Remove step 1"}))
        fireEvent.click(screen.getByRole("button", {name: "Add a step"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        const added = within(screen.getByRole("group", {name: "Step 3"}))
        fireEvent.change(added.getByLabelText("From % of the map"), {target: {value: "40"}})
        fireEvent.change(added.getByLabelText("Times slower"), {target: {value: "6"}})
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.bonuses?.quiz?.enabled).toBe(false)
        expect(sent(admin)?.rules?.toll?.steps.map(step => [step.share, step.slowdown])).toEqual([[0.2, 2.5], [0.3, 4], [0.4, 6]])
    })

    it("freezes a set: the map takes no play", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        fireEvent.click(screen.getByLabelText("Frozen: the map takes no play"))
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.frozen).toBe(true)
    })

    it("makes a new set from a copy of another", async () => {
        const admin = new FakeAdmin()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        expect(screen.getByRole("region", {name: "New set from default"})).toBeTruthy()
        expect((screen.getByLabelText("Name") as HTMLInputElement).value).toBe("")
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        type("Name", "rush")
        type("Refill, times as fast", "3")
        save()

        await waitFor(() => expect(rowOf("rush")).toBeTruthy())
        expect(sent(admin)?.rules?.bonuses?.kinds?.refill).toBe(5)
    })

    it("quotes the server's refusal and keeps the form", async () => {
        const admin = withCalm()
        admin.refuse("saveRuleSet", Code.InvalidArgument)
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        save()

        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
        expect(screen.getByRole("region", {name: "Edit calm"})).toBeTruthy()
    })

    it("says why a save of the set that plays now is refused", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveRuleSet", Code.FailedPrecondition)
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        type("Name", "default")
        save()

        expect((await screen.findByRole("alert")).textContent).toContain("plays now")
    })
})
