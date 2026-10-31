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

    it("edits the set that plays, and says it takes over at once", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        type("New account, clicks in hand", "60")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText("It plays now: it takes over at once.")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(sent(admin)?.name).toBe("default"))
        expect(sent(admin)?.rules?.clicks?.newAccountClicks).toBe(60)
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

    it("asks to save with what changes, each value changed saying what it was", async () => {
        render(<RulesPanel planet={withCalm()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        expect(screen.getByText("No change yet")).toBeTruthy()
        type("Shortest box wait (s)", "120")
        type("Clicks in hand", "60.0")
        fireEvent.click(screen.getByLabelText("Frozen: the map takes no play"))
        expect(screen.getByText("2 changes")).toBeTruthy()
        expect(screen.getByText("2 min · was 4 min")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Save"}))

        const changes = within(screen.getByRole("list", {name: "What changes"})).getAllByRole("listitem")
        expect(changes.map(change => change.textContent)).toEqual([
            "Frozen no → yes",
            "Boxes · Shortest box wait (s) 4 min → 2 min",
        ])
        expect(screen.queryByText("It plays now: it takes over at once.")).toBeNull()
    })

    it("switches quizzes and the sign-in refill off, and edits the toll", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        fireEvent.click(screen.getByLabelText("Quizzes on"))
        fireEvent.click(screen.getByLabelText("Refill at first sign-in"))
        fireEvent.click(screen.getByRole("button", {name: "Remove step 1"}))
        fireEvent.click(screen.getByRole("button", {name: "Add a step"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        const added = within(screen.getByRole("group", {name: "Step 3"}))
        fireEvent.change(added.getByLabelText("From % of the map"), {target: {value: "40"}})
        fireEvent.change(added.getByLabelText("Times slower"), {target: {value: "6"}})
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.bonuses?.quiz?.enabled).toBe(false)
        expect(sent(admin)?.rules?.bonuses?.signInRefill).toBe(false)
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

    it("locks the flags of a set: no player changes country", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        fireEvent.click(screen.getByLabelText("Flags locked: no player changes country"))
        expect(screen.getByText("1 change")).toBeTruthy()
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.allegianceLocked).toBe(true)
        expect(sent(admin)?.rules?.frozen).toBe(false)
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
        type("Refill, times as fast", "3")
        save()

        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
        expect(screen.getByRole("region", {name: "Edit calm"})).toBeTruthy()
    })
})
