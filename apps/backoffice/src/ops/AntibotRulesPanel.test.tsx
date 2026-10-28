// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {AntibotRulesPanel} from "./AntibotRulesPanel.tsx"

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

// default plays; calm does not, with its defender on.
function withCalm(): FakeAdmin {
    const admin = new FakeAdmin()
    const calm = new RuleSet({name: "calm", rules: admin.antibotRules.sets[0].rules?.clone()})
    calm.rules!.bans!.enforce = false
    admin.antibotRules = new GetRulesResponse({sets: [admin.antibotRules.sets[0], calm], playing: "default"})
    return admin
}

describe("the antibot rules page", () => {
    it("lists every set, the one playing marked", async () => {
        render(<AntibotRulesPanel antibot={new FakeAdmin().antibot}/>)
        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.getByText("default", {selector: "code"})).toBeTruthy()
        const row = rowOf("default")
        expect(row.getAttribute("aria-current")).toBe("true")
        expect(within(row).getByText("Playing now")).toBeTruthy()
        expect(within(row).getByText("Enforced")).toBeTruthy()
        expect(within(row).getByText("6 scopes")).toBeTruthy()
        expect(within(row).getByText("defender")).toBeTruthy()
    })

    it("offers no edit of the set that plays, only a copy", async () => {
        render(<AntibotRulesPanel antibot={withCalm().antibot}/>)
        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.queryByRole("button", {name: "Edit default"})).toBeNull()
        expect(screen.getByRole("button", {name: "Edit calm"})).toBeTruthy()
    })

    it("edits a set that does not play, asks with what changes, and reads the rules again", async () => {
        const admin = withCalm()
        render(<AntibotRulesPanel antibot={admin.antibot}/>)
        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        expect(saveButton().disabled).toBe(true)
        fireEvent.click(screen.getByLabelText("Defender · Enabled"))
        type("Bans · Reflag interval", "600")
        expect(screen.getByText("2 changes")).toBeTruthy()
        expect(screen.getByText("10 min · was 5 min")).toBeTruthy()
        fireEvent.click(saveButton())
        const changes = within(screen.getByRole("list", {name: "What changes"})).getAllByRole("listitem")
        expect(changes.map(change => change.textContent)).toEqual([
            "Bans · Reflag interval 5 min → 10 min",
            "Defender · Enabled off → on",
        ])
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))
        await waitFor(() => expect(within(rowOf("calm")).getByText("None")).toBeTruthy())
        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.name).toBe("calm")
        expect(asked.set.rules?.defender?.enabled).toBe(true)
        expect(Number(asked.set.rules?.bans?.reflagInterval?.seconds)).toBe(600)
        expect(asked.set.rules?.cohort?.certainMembers).toBe(6)
        expect(screen.queryByRole("region", {name: "Edit calm"})).toBeNull()
    })

    it("edits the ban ladder step by step", async () => {
        const admin = withCalm()
        render(<AntibotRulesPanel antibot={admin.antibot}/>)
        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        fireEvent.click(screen.getByRole("button", {name: "Remove ban durations 3"}))
        fireEvent.click(screen.getByRole("button", {name: "Add a step"}))
        expect(saveButton().disabled).toBe(true)
        expect(screen.getByRole("alert").textContent).toBe("Bans · Ban durations: a step is empty.")
        type("Bans · Ban durations 3", "2592000")
        save()
        await waitFor(() => expect(admin.askedFor("saveRuleSet")).toHaveLength(1))
        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.rules?.bans?.banDurations.map(duration => Number(duration.seconds))).toEqual([86400, 604800, 2592000])
    })

    it("makes a new set from a copy of the one playing", async () => {
        render(<AntibotRulesPanel antibot={new FakeAdmin().antibot}/>)
        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        expect(screen.getByRole("region", {name: "New set from default"})).toBeTruthy()
        expect(saveButton().disabled).toBe(true)
        type("Name", "rush")
        type("Cohort · Certain members", "50")
        save()
        await waitFor(() => expect(within(rowOf("rush")).getByText("50 scopes")).toBeTruthy())
    })

    it("sends nothing while a value is no number, and says which", async () => {
        const admin = withCalm()
        render(<AntibotRulesPanel antibot={admin.antibot}/>)
        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        type("Cohort · Certain members", "")
        expect(screen.getByRole("alert").textContent).toBe("Cohort · Certain members: a number is needed.")
        expect(saveButton().disabled).toBe(true)
        expect(admin.askedFor("saveRuleSet")).toEqual([])
    })

    it("quotes the server's refusal and keeps the form", async () => {
        const admin = withCalm()
        admin.refuse("saveRuleSet", Code.InvalidArgument)
        render(<AntibotRulesPanel antibot={admin.antibot}/>)
        fireEvent.click(await screen.findByRole("button", {name: "Edit calm"}))
        type("Cohort · Certain members", "50")
        save()
        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
        expect(screen.getByRole("region", {name: "Edit calm"})).toBeTruthy()
    })

    it("says a name that plays now cannot be saved", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveRuleSet", Code.FailedPrecondition)
        render(<AntibotRulesPanel antibot={admin.antibot}/>)
        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        type("Name", "rush")
        save()
        expect((await screen.findByRole("alert")).textContent)
            .toBe("A set of this name plays now. Save it under another name.")
    })
})
