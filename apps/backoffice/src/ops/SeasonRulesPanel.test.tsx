// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {RuleSet} from "../gen/grpc/seasons/v1/admin_pb.ts"
import {SeasonRulesPanel} from "./SeasonRulesPanel.tsx"

afterEach(cleanup)

const rowOf = (set: string) =>
    screen.getAllByRole("row").find(row => row.querySelector("td")?.textContent === set) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const save = () => {
    fireEvent.click(screen.getByRole("button", {name: "Save"}))
    fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))
}

describe("the season rules page", () => {
    it("lists every set with its round, the one playing marked", async () => {
        render(<SeasonRulesPanel seasons={new FakeAdmin().seasons}/>)

        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.getByText("default", {selector: "code"})).toBeTruthy()
        expect(rowOf("default").getAttribute("aria-current")).toBe("true")
        expect(within(rowOf("default")).getByText("Days only")).toBeTruthy()
        expect(within(rowOf("rush")).getByText("Of its own, 3× a day's points")).toBeTruthy()
    })

    it("offers no edit of the set that plays, only a copy", async () => {
        render(<SeasonRulesPanel seasons={new FakeAdmin().seasons}/>)

        await screen.findByRole("button", {name: "Copy default"})
        expect(screen.queryByRole("button", {name: "Edit default"})).toBeNull()
        expect(screen.getByRole("button", {name: "Edit rush"})).toBeTruthy()
    })

    it("edits a set that does not play once confirmed, and reads the rules again", async () => {
        const admin = new FakeAdmin()
        render(<SeasonRulesPanel seasons={admin.seasons}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit rush"}))
        expect((screen.getByLabelText("Name") as HTMLInputElement).disabled).toBe(true)
        type("Points ×", "5")
        save()

        await waitFor(() => expect(within(rowOf("rush")).getByText("Of its own, 5× a day's points")).toBeTruthy())
        const [asked] = admin.askedFor("saveRuleSet") as {set: RuleSet}[]
        expect(asked.set.name).toBe("rush")
        expect(asked.set.rules?.ownRound?.enabled).toBe(true)
        expect(asked.set.rules?.ownRound?.times).toBe(5)
    })

    it("makes a battle's set from a copy of default", async () => {
        const admin = new FakeAdmin()
        render(<SeasonRulesPanel seasons={admin.seasons}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        type("Name", "urf")
        fireEvent.click(screen.getByLabelText("A round of its own"))
        type("Points ×", "3")
        save()

        await waitFor(() => expect(within(rowOf("urf")).getByText("Of its own, 3× a day's points")).toBeTruthy())
    })

    it("sends nothing while the points are not 1 to 10", async () => {
        const admin = new FakeAdmin()
        render(<SeasonRulesPanel seasons={admin.seasons}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Edit rush"}))
        for (const times of ["0", "11", "2.5", ""]) {
            type("Points ×", times)
            expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        }
        expect(admin.askedFor("saveRuleSet")).toEqual([])
    })

    it("says a name that plays now cannot be saved", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveRuleSet", Code.FailedPrecondition)
        render(<SeasonRulesPanel seasons={admin.seasons}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Copy default"}))
        type("Name", "urf")
        save()

        expect((await screen.findByRole("alert")).textContent)
            .toBe("A set of this name plays now. Save it under another name.")
    })
})
