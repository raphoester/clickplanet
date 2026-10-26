// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetRulesResponse as AntibotRules, RuleSet as AntibotRuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {AddEntryRequest, EditEntryRequest} from "../gen/grpc/rules/v1/admin_pb.ts"
import {SchedulePanel} from "./SchedulePanel.tsx"

afterEach(cleanup)

const rowOf = (set: string) =>
    screen.getAllByRole("row").find(row => row.querySelector("strong")?.textContent === set) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

// Planet and the antibot keep calm and frozen too, and play calm.
function keeping(): FakeAdmin {
    const admin = new FakeAdmin()
    const planetDefault = admin.rules.sets[0]
    admin.rules = new GetRulesResponse({
        sets: ["default", "calm", "frozen", "rush"].map(name => new RuleSet({name, rules: planetDefault.rules?.clone()})),
        playing: "calm",
    })
    const antibotDefault = admin.antibotRules.sets[0]
    admin.antibotRules = new AntibotRules({
        sets: ["default", "calm", "frozen", "rush"].map(name => new AntibotRuleSet({name, rules: antibotDefault.rules?.clone()})),
        playing: "calm",
    })
    return admin
}

function show(admin: FakeAdmin) {
    render(<SchedulePanel rules={admin.ruleSchedule} planet={admin} antibot={admin.antibot}/>)
}

describe("the rule schedule page", () => {
    it("lists every entry by start, the one playing marked, the ended one muted, and what each module plays", async () => {
        show(keeping())

        await screen.findByRole("table")
        const sets = screen.getAllByRole("row").slice(1).map(row => row.querySelector("strong")?.textContent)
        expect(sets).toEqual(["rush", "calm", "frozen"])
        expect(rowOf("calm").getAttribute("aria-current")).toBe("true")
        expect(rowOf("rush").className).toBe("muted")
        expect(within(rowOf("frozen")).getByText("2")).toBeTruthy()
        expect(screen.getByText(/the antibot plays/)).toBeTruthy()
        expect(screen.queryByRole("alert")).toBeNull()
    })

    it("offers a change only of an entry to come", async () => {
        show(keeping())

        await screen.findByRole("table")
        expect(within(rowOf("frozen")).getByRole("button", {name: "Edit"})).toBeTruthy()
        expect(within(rowOf("frozen")).getByRole("button", {name: "Delete"})).toBeTruthy()
        expect(within(rowOf("calm")).queryByRole("button")).toBeNull()
        expect(within(rowOf("rush")).queryByRole("button")).toBeNull()
    })

    it("warns of a set a module does not keep, on an entry that has not ended", async () => {
        show(new FakeAdmin())

        await screen.findByRole("table")
        expect(within(rowOf("frozen")).getByText("No set frozen in planet or the antibot.")).toBeTruthy()
        expect(within(rowOf("calm")).getByText("No set calm in planet or the antibot.")).toBeTruthy()
        expect(within(rowOf("rush")).queryByRole("alert")).toBeNull()
    })

    it("says when a module does not play what the schedule plays", async () => {
        show(new FakeAdmin())

        expect(await screen.findByText(/Planet does not play/)).toBeTruthy()
        expect(screen.getByText(/The antibot does not play/)).toBeTruthy()
    })

    it("adds an entry once confirmed, and reads the schedule again", async () => {
        const admin = keeping()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add an entry"}))
        type("Set", "rush")
        type("Starts", "2099-10-31T20:00")
        type("Ends", "2099-10-31T23:30")
        type("Priority", "3")
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        await waitFor(() => expect(screen.queryByRole("region", {name: "New entry"})).toBeNull())
        const [asked] = admin.askedFor("addEntry") as AddEntryRequest[]
        expect(asked.ruleSet).toBe("rush")
        expect(asked.startsAt?.toDate()).toEqual(new Date("2099-10-31T20:00"))
        expect(asked.endsAt?.toDate()).toEqual(new Date("2099-10-31T23:30"))
        expect(asked.priority).toBe(3)
        expect(admin.askedFor("getSchedule")).toHaveLength(2)
    })

    it("offers only the sets both modules keep", async () => {
        const admin = keeping()
        admin.antibotRules = new AntibotRules({sets: [admin.antibotRules.sets[0], admin.antibotRules.sets[3]], playing: "calm"})
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add an entry"}))

        const options = within(screen.getByLabelText("Set")).getAllByRole("option").map(option => option.textContent)
        expect(options).toEqual(["default", "rush"])
    })

    it("edits an entry to come once confirmed", async () => {
        const admin = keeping()
        show(admin)

        fireEvent.click(within(await screen.findByRole("table")).getByRole("button", {name: "Edit"}))
        expect((screen.getByLabelText("Priority") as HTMLInputElement).value).toBe("2")
        type("Set", "calm")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editEntry")).toHaveLength(1))
        const [asked] = admin.askedFor("editEntry") as EditEntryRequest[]
        expect(asked.id).toBe("entry-to-come")
        expect(asked.ruleSet).toBe("calm")
        expect(asked.priority).toBe(2)
    })

    it("deletes an entry to come once confirmed", async () => {
        const admin = keeping()
        show(admin)

        fireEvent.click(within(await screen.findByRole("table")).getByRole("button", {name: "Delete"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, delete"}))

        await waitFor(() => expect(screen.queryByText("frozen")).toBeNull())
        expect(admin.askedFor("deleteEntry")).toEqual([{id: "entry-to-come"}])
    })

    it("quotes the server when it refuses an entry", async () => {
        const admin = keeping()
        admin.refuse("addEntry", Code.FailedPrecondition)
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add an entry"}))
        type("Starts", "2099-10-31T20:00")
        type("Ends", "2099-10-31T23:30")
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()
        expect(screen.getByRole("region", {name: "New entry"})).toBeTruthy()
    })
})
