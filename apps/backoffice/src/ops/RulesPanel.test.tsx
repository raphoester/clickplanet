// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
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

const pick = (label: string, value: "default" | "yes" | "no") =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const BOMBS_OFF = "Bombs off: no box offers one, and none is dropped"
const ENCLOSE_OFF = "Enclose off: no box offers one, and none is used"

// A set that holds only a slower refill, and plays default's for the rest.
function withSlow(): FakeAdmin {
    const admin = new FakeAdmin()
    const slow = new RuleSet({name: "slow", rules: Rules.fromJson({clicks: {tempo: 0.5}})})
    admin.rules = new GetRulesResponse({sets: [admin.rules.sets[0], slow], playing: "default"})
    return admin
}

// A second set, kept beside default and not playing, that holds every value.
function withCalm(): FakeAdmin {
    const admin = new FakeAdmin()
    const calm = new RuleSet({name: "calm", rules: admin.rules.sets[0].rules?.clone()})
    admin.rules = new GetRulesResponse({sets: [admin.rules.sets[0], calm], playing: "default"})
    return admin
}

describe("the game rules page", () => {
    it("lists every set with its refill and its clicks, and says which plays", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.getByText("Playing:").textContent).toContain("default")
        const row = rowOf("default")
        expect(row.getAttribute("aria-current")).toBe("true")
        expect(within(row).getByText("1×")).toBeTruthy()
        expect(within(row).getByText("60 in hand, 0.2 a second")).toBeTruthy()
    })

    it("edits a hand's charges, and leaves the boxes and quizzes to their own page", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        const charges = within(screen.getByRole("group", {name: "Charges"}))
        expect(charges.getAllByRole("spinbutton").map(field => field.closest("label")?.textContent)).toEqual([
            "Spread clicks held", "Spread rings, 1 to 3", "Bomb rings", "Enclose tiles, at most", "Enclosures held", "Shields held",
            "Shields on a tile, at most",
        ])
        expect(screen.queryByLabelText("Shortest box wait (s)")).toBeNull()
        expect(screen.queryByLabelText("Quizzes on")).toBeNull()
        expect(screen.queryByLabelText("Refill at first sign-in")).toBeNull()
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
        type("Enclosures held", "5")
        save()

        await waitFor(() => expect(within(rowOf("calm")).getByText("3×")).toBeTruthy())
        const rules = sent(admin)?.rules
        expect(sent(admin)?.name).toBe("calm")
        expect(rules?.clicks?.tempo).toBe(3)
        expect(rules?.charges?.shield?.perTile).toBe(20)
        expect(rules?.charges?.enclose?.held).toBe(5)
        expect(rules?.clicks?.burst).toBe(60)
        expect(screen.queryByRole("region", {name: "Edit calm"})).toBeNull()
    })

    it("asks to save with what changes, each value changed saying what it was", async () => {
        render(<RulesPanel planet={withCalm()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        expect(screen.getByText("No change yet")).toBeTruthy()
        type("New account, signed-in pace for (s)", "600")
        type("Clicks in hand", "60.0")
        pick("Frozen: the map takes no play", "yes")
        expect(screen.getByText("2 changes")).toBeTruthy()
        expect(screen.getByText("10 min · was 30 min")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Save"}))

        const changes = within(screen.getByRole("list", {name: "What changes"})).getAllByRole("listitem")
        expect(changes.map(change => change.textContent)).toEqual([
            "Frozen no → yes",
            "Clicks · New account, signed-in pace for (s) 30 min → 10 min",
        ])
        expect(screen.queryByText("It plays now: it takes over at once.")).toBeNull()
    })

    it("switches unlimited enclosures on, and edits the toll", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick("Enclosures unlimited: enclose spends no charge", "yes")
        fireEvent.click(screen.getByRole("button", {name: "Remove step 1"}))
        fireEvent.click(screen.getByRole("button", {name: "Add a step"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        const added = within(screen.getByRole("group", {name: "Step 3"}))
        fireEvent.change(added.getByLabelText("From % of the map"), {target: {value: "40"}})
        fireEvent.change(added.getByLabelText("Times slower"), {target: {value: "6"}})
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.charges?.enclose?.unlimited).toBe(true)
        expect(sent(admin)?.rules?.toll?.steps.map(step => [step.share, step.slowdown])).toEqual([[0.2, 2.5], [0.3, 4], [0.4, 6]])
    })

    it("freezes a set: the map takes no play", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick("Frozen: the map takes no play", "yes")
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.frozen).toBe(true)
    })

    it("locks the flags of a set: no player changes country", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick("Flags locked: no player changes country", "yes")
        expect(screen.getByText("1 change")).toBeTruthy()
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.allegianceLocked).toBe(true)
        expect(sent(admin)?.rules?.frozen).toBe(false)
    })

    it("plays a set as an arena battle, a switch default never shows", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick("Arena: the battle is fought on a fresh map", "yes")
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.arena?.on).toBe(true)
    })

    it("plays the reach, the knock-out and fixed capital shields, each a switch of its own", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick("Reach: a take lands only next to the flag's land or across a bridge", "yes")
        pick("Knock-out: a capital taken clears its flag", "yes")
        pick("No shield on a capital", "no")
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.reach).toBe(true)
        expect(sent(admin)?.rules?.capitals?.knockOut).toBe(true)
        expect(sent(admin)?.rules?.capitals?.fixedShields).toBe(false)
    })

    it("turns bombs off with a switch of its own", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick(BOMBS_OFF, "yes")
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.charges?.bomb?.off).toBe(true)
    })

    it("turns enclose off with a switch of its own, and keeps every other value", async () => {
        const admin = withCalm()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick(ENCLOSE_OFF, "yes")
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.charges?.enclose?.off).toBe(true)
        expect(sent(admin)?.rules?.charges?.enclose?.maxTiles).toBe(25)
        expect(sent(admin)?.rules?.charges?.bomb?.off).toBe(false)
    })

    it("makes a new set from a copy of default, which holds only what is typed", async () => {
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
        expect(sent(admin)?.rules?.toJson()).toEqual({clicks: {tempo: 3}})
        expect(within(rowOf("rush")).getByText("3×")).toBeTruthy()
        expect(within(rowOf("rush")).getByText("60 in hand, 0.2 a second")).toBeTruthy()
    })

    it("shows each set as it plays, default's values where it holds none", async () => {
        render(<RulesPanel planet={withSlow()}/>)

        await screen.findByRole("button", {name: "Edit slow"})
        expect(within(rowOf("slow")).getByText("0.5×")).toBeTruthy()
        expect(within(rowOf("slow")).getByText("60 in hand, 0.2 a second")).toBeTruthy()
    })

    it("shows default's value in a field the set leaves empty, and sends only what the set holds", async () => {
        const admin = withSlow()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit slow"}))
        expect((screen.getByLabelText("Clicks in hand") as HTMLInputElement).value).toBe("")
        expect(screen.getByText("default: 60")).toBeTruthy()
        expect((screen.getByLabelText(BOMBS_OFF) as HTMLSelectElement).value).toBe("default")
        expect(screen.getByText("Default's: 10% ×1.5, 20% ×2.5, 30% ×4")).toBeTruthy()
        type("Clicks in hand", "40")
        pick(BOMBS_OFF, "yes")
        fireEvent.click(screen.getByRole("button", {name: "Refill, times as fast: use default"}))
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.toJson()).toEqual({clicks: {burst: 40}, charges: {bomb: {off: true}}})
    })

    it("gives a set a toll of its own, none at all included", async () => {
        const admin = withSlow()
        render(<RulesPanel planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit slow"}))
        fireEvent.click(screen.getByLabelText("A toll of its own"))
        fireEvent.click(screen.getByRole("button", {name: "Remove step 3"}))
        fireEvent.click(screen.getByRole("button", {name: "Remove step 2"}))
        fireEvent.click(screen.getByRole("button", {name: "Remove step 1"}))
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.toll?.steps).toEqual([])
        expect(sent(admin)?.rules?.toll).toBeDefined()
    })

    it("edits default with a box for each switch and every value needed", async () => {
        render(<RulesPanel planet={new FakeAdmin()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        expect((screen.getByLabelText(BOMBS_OFF) as HTMLInputElement).type).toBe("checkbox")
        expect(screen.queryByLabelText("A toll of its own")).toBeNull()
        type("Clicks in hand", "")
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
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
