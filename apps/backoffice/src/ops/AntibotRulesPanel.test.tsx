// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {AntibotRulesPanel} from "./AntibotRulesPanel.tsx"

afterEach(cleanup)

const rowOf = (set: string) =>
    screen.getAllByRole("row").find(row => row.querySelector("td")?.textContent === set) as HTMLElement

const rulesText = () => screen.getByLabelText("Rules (JSON)") as HTMLTextAreaElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const save = () => {
    fireEvent.click(screen.getByRole("button", {name: "Save"}))
    fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))
}

describe("the antibot rules page", () => {
    it("lists every set, the one playing marked", async () => {
        render(<AntibotRulesPanel antibot={new FakeAdmin()}/>)

        await screen.findByRole("button", {name: "Edit default"})
        expect(screen.getByText("default", {selector: "code"})).toBeTruthy()
        const row = rowOf("default")
        expect(row.getAttribute("aria-current")).toBe("true")
        expect(within(row).getByText("Enforced")).toBeTruthy()
        expect(within(row).getByText("6 scopes")).toBeTruthy()
        expect(within(row).getByText("defender")).toBeTruthy()
    })

    it("edits a set once confirmed, and reads the rules again", async () => {
        const admin = new FakeAdmin()
        render(<AntibotRulesPanel antibot={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        type("Rules (JSON)", rulesText().value.replace("\"enforce\": true", "\"enforce\": false"))
        save()

        await waitFor(() => expect(within(rowOf("default")).getByText("Not enforced")).toBeTruthy())
        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.name).toBe("default")
        expect(asked.set.rules?.bans?.enforce).toBe(false)
        expect(screen.queryByRole("region", {name: "Edit default"})).toBeNull()
    })

    it("turns a watchdog back on", async () => {
        const admin = new FakeAdmin()
        render(<AntibotRulesPanel antibot={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        const text = rulesText().value
        type("Rules (JSON)", text.replace(/("defender": \{\s*"enabled": )false/, "$1true"))
        save()

        await waitFor(() => expect(within(rowOf("default")).getByText("None")).toBeTruthy())
    })

    it("makes a new set from a copy of default", async () => {
        render(<AntibotRulesPanel antibot={new FakeAdmin()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "New set"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        type("Name", "rush")
        type("Rules (JSON)", rulesText().value.replace("\"certainMembers\": 6", "\"certainMembers\": 50"))
        save()

        await waitFor(() => expect(within(rowOf("rush")).getByText("50 scopes")).toBeTruthy())
    })

    it("sends nothing while the text is no set, and says why", async () => {
        const admin = new FakeAdmin()
        render(<AntibotRulesPanel antibot={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        type("Rules (JSON)", "{\"cohort\": {\"crowd\": 50}}")

        expect(screen.getByRole("alert").textContent).toContain("crowd")
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        expect(admin.askedFor("saveRuleSet")).toEqual([])
    })

    it("quotes the server's refusal and keeps the form", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveRuleSet", Code.InvalidArgument)
        render(<AntibotRulesPanel antibot={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit default"}))
        save()

        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
        expect(screen.getByRole("region", {name: "Edit default"})).toBeTruthy()
    })
})
