// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import {BonusRulesPanel} from "./BonusRulesPanel.tsx"

afterEach(cleanup)

const rowOf = (set: string) =>
    screen.getAllByRole("row").find(row => row.querySelector("td")?.textContent === set) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const saveButton = () => screen.getByRole("button", {name: "Save"}) as HTMLButtonElement

const save = () => {
    fireEvent.click(saveButton())
    fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))
}

const sent = (admin: FakeAdmin) => (admin.askedFor("saveBonusRuleSet") as {set: RuleSet}[]).at(-1)?.set

const pick = (label: string, value: "default" | "yes" | "no") =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const FROZEN = "Frozen: no box or quiz is offered, caught or answered"

const show = (admin: FakeAdmin) => render(<BonusRulesPanel bonuses={admin.bonuses}/>)

// A set that holds only fewer boxes an hour and no quiz, and plays default's for the rest.
function withQuiet(): FakeAdmin {
    const admin = new FakeAdmin()
    const quiet = new RuleSet({name: "quiet", rules: Rules.fromJson({boxes: {maxChargesPerHour: 6}, quiz: {enabled: false}})})
    admin.bonusRules = new GetRulesResponse({sets: [admin.bonusRules.sets[0], quiet], playing: "default"})
    return admin
}

// A second set, kept beside default and not playing, that holds every value.
function withCalm(): FakeAdmin {
    const admin = new FakeAdmin()
    const calm = new RuleSet({name: "calm", rules: admin.bonusRules.sets[0].rules?.clone()})
    admin.bonusRules = new GetRulesResponse({sets: [admin.bonusRules.sets[0], calm], playing: "default"})
    return admin
}

describe("the box and quiz rules page", () => {
    it("lists every set with its boxes and its quizzes, and says which plays", async () => {
        show(new FakeAdmin())

        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.getByText("default", {selector: "code"})).toBeTruthy()
        const row = rowOf("default")
        expect(row.getAttribute("aria-current")).toBe("true")
        expect(within(row).getByText("Playing now")).toBeTruthy()
        expect(within(row).getByText("Open")).toBeTruthy()
        expect(within(row).getByText("Every 4 min to 8 min, 12 an hour at most")).toBeTruthy()
        expect(within(row).getByText("Every 4 min to 8 min, 12 won an hour at most")).toBeTruthy()
    })

    it("shows each set as it plays, default's values where it holds none", async () => {
        show(withQuiet())

        await screen.findByRole("button", {name: "Edit quiet"})
        expect(within(rowOf("quiet")).getByText("Every 4 min to 8 min, 6 an hour at most")).toBeTruthy()
        expect(within(rowOf("quiet")).getByText("Off")).toBeTruthy()
        expect(within(rowOf("quiet")).getByText("Open")).toBeTruthy()
    })

    it("edits the set that plays, and says it takes over at once", async () => {
        const admin = withCalm()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        type("Box flight (s)", "20")
        fireEvent.click(saveButton())
        expect(screen.getByText("It plays now: it takes over at once.")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(sent(admin)?.name).toBe("default"))
        expect(Number(sent(admin)?.rules?.offerTtl?.seconds)).toBe(20)
        expect(sent(admin)?.rules?.boxes?.maxChargesPerHour).toBe(12)
    })

    it("edits a set once confirmed, asks with what changes, and reads the rules again", async () => {
        const admin = withCalm()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        expect(saveButton().disabled).toBe(true)
        expect(screen.getByText("No change yet")).toBeTruthy()
        type("Shortest box wait (s)", "120")
        type("Charges an hour, at most", "30")
        type("Bomb, weight", "1.0")
        type("Shields a box, at most", "25")
        type("Time to answer (s)", "6")
        expect(screen.getByText("4 changes")).toBeTruthy()
        expect(screen.getByText("2 min · was 4 min")).toBeTruthy()
        fireEvent.click(saveButton())

        const changes = within(screen.getByRole("list", {name: "What changes"})).getAllByRole("listitem")
        expect(changes.map(change => change.textContent)).toEqual([
            "Boxes · Shortest box wait (s) 4 min → 2 min",
            "Boxes · Charges an hour, at most 12 → 30",
            "What it grants · Shields a box, at most 20 → 25",
            "Quizzes · Time to answer (s) 8 s → 6 s",
        ])
        expect(screen.queryByText("It plays now: it takes over at once.")).toBeNull()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(within(rowOf("calm")).getByText("Every 2 min to 8 min, 30 an hour at most")).toBeTruthy())
        const rules = sent(admin)?.rules
        expect(sent(admin)?.name).toBe("calm")
        expect(Number(rules?.boxes?.minInterval?.seconds)).toBe(120)
        expect(rules?.shield?.maxPerBox).toBe(25)
        expect(Number(rules?.quiz?.answerWindow?.seconds)).toBe(6)
        expect(rules?.kinds?.refill).toBe(5)
        expect(screen.queryByRole("region", {name: "Edit calm"})).toBeNull()
    })

    it("freezes a set, and switches quizzes and the sign-in refill off", async () => {
        const admin = withCalm()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        pick(FROZEN, "yes")
        pick("Quizzes on", "no")
        pick("Refill at first sign-in", "no")
        expect(screen.getByText("3 changes")).toBeTruthy()
        save()

        await waitFor(() => expect(within(rowOf("calm")).getByText("Frozen")).toBeTruthy())
        expect(within(rowOf("calm")).getByText("Off")).toBeTruthy()
        expect(sent(admin)?.rules?.frozen).toBe(true)
        expect(sent(admin)?.rules?.quiz?.enabled).toBe(false)
        expect(sent(admin)?.rules?.signInRefill).toBe(false)
    })

    it("makes a new set from a copy of default, which holds only what is typed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        expect(screen.getByRole("region", {name: "New set from default"})).toBeTruthy()
        expect((screen.getByLabelText("Name") as HTMLInputElement).value).toBe("")
        expect(saveButton().disabled).toBe(true)
        type("Name", "rush")
        type("Longest box wait (s)", "60")
        type("Shortest box wait (s)", "60")
        save()

        await waitFor(() => expect(rowOf("rush")).toBeTruthy())
        expect(sent(admin)?.rules?.toJson()).toEqual({boxes: {minInterval: "60s", maxInterval: "60s"}})
        expect(within(rowOf("rush")).getByText("Every 1 min, 12 an hour at most")).toBeTruthy()
    })

    it("shows default's value in a field the set leaves empty, and sends only what the set holds", async () => {
        const admin = withQuiet()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Edit quiet"}))
        expect((screen.getByLabelText("Shortest box wait (s)") as HTMLInputElement).value).toBe("")
        expect((screen.getByLabelText("Shortest box wait (s)") as HTMLInputElement).placeholder).toBe("240")
        expect(screen.getAllByText("default: 4 min").length).toBeGreaterThan(0)
        expect((screen.getByLabelText(FROZEN) as HTMLSelectElement).value).toBe("default")
        expect((screen.getByLabelText("Quizzes on") as HTMLSelectElement).value).toBe("no")
        type("Spread clicks a box, at most", "0")
        pick("Quizzes on", "default")
        fireEvent.click(screen.getByRole("button", {name: "Charges an hour, at most: use default"}))
        save()

        await waitFor(() => expect(sent(admin)).toBeDefined())
        expect(sent(admin)?.rules?.toJson()).toEqual({spread: {maxPerBox: 0}})
    })

    it("edits default with a box for each switch and every value needed", async () => {
        show(new FakeAdmin())

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        expect((screen.getByLabelText(FROZEN) as HTMLInputElement).type).toBe("checkbox")
        expect((screen.getByLabelText("Quizzes on") as HTMLInputElement).checked).toBe(true)
        expect(screen.queryByText("An empty value plays default's.")).toBeNull()
        type("Banner (s)", "")
        expect(saveButton().disabled).toBe(true)
        type("Banner (s)", "a while")
        expect(saveButton().disabled).toBe(true)
    })

    it("quotes the server's refusal and keeps the form", async () => {
        const admin = withCalm()
        admin.refuse("saveBonusRuleSet", Code.InvalidArgument)
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        type("Shields a box, at least", "50")
        save()

        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
        expect(screen.getByRole("region", {name: "Edit calm"})).toBeTruthy()
    })

    it("says a read that failed, and reads again", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getBonusRules", Code.Unavailable)
        show(admin)

        const again = await screen.findByRole("button", {name: "Try again"})
        admin.refusals.delete("getBonusRules")
        fireEvent.click(again)

        expect(await screen.findByRole("button", {name: "Copy default"})).toBeTruthy()
    })
})
