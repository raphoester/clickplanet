// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {AddSeasonRequest, EditSeasonRequest} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {SeasonsPanel} from "./SeasonsPanel.tsx"

afterEach(cleanup)

const rowOf = (number: number) =>
    screen.getAllByRole("row").find(row => row.querySelector("strong")?.textContent === String(number)) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

function show(admin: FakeAdmin) {
    render(<SeasonsPanel gamemaster={admin.gamemaster}/>)
}

describe("the seasons page", () => {
    it("lists every season by number, the running one marked, the done one muted", async () => {
        show(new FakeAdmin())

        await screen.findByRole("table")
        const numbers = screen.getAllByRole("row").slice(1).map(row => row.querySelector("strong")?.textContent)
        expect(numbers).toEqual(["1", "2", "3"])
        expect(within(rowOf(1)).getByText("done")).toBeTruthy()
        expect(rowOf(1).className).toBe("muted")
        expect(within(rowOf(2)).getByText("running")).toBeTruthy()
        expect(rowOf(2).getAttribute("aria-current")).toBe("true")
        expect(within(rowOf(3)).getByText("to come")).toBeTruthy()
    })

    it("offers an edit of a season that has not ended, and a delete of the last one to come", async () => {
        show(new FakeAdmin())

        await screen.findByRole("table")
        expect(within(rowOf(1)).queryByRole("button")).toBeNull()
        expect(within(rowOf(2)).getByRole("button", {name: "Edit"})).toBeTruthy()
        expect(within(rowOf(2)).queryByRole("button", {name: "Delete"})).toBeNull()
        expect(within(rowOf(3)).getByRole("button", {name: "Edit"})).toBeTruthy()
        expect(within(rowOf(3)).getByRole("button", {name: "Delete"})).toBeTruthy()
    })

    it("adds a season once confirmed, starting where the last one ends, and reads the seasons again", async () => {
        const admin = new FakeAdmin()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        expect((screen.getByLabelText("Starts") as HTMLInputElement).value).not.toBe("")
        type("Starts", "2099-10-01T00:00")
        type("Ends", "2099-12-31T23:30")
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        expect(screen.getByText(/^Add a season from /)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        await waitFor(() => expect(screen.queryByRole("region", {name: "New season"})).toBeNull())
        const [asked] = admin.askedFor("addSeason") as AddSeasonRequest[]
        expect(asked.startsAt?.toDate()).toEqual(new Date("2099-10-01T00:00"))
        expect(asked.endsAt?.toDate()).toEqual(new Date("2099-12-31T23:30"))
        expect(admin.askedFor("getSeasons")).toHaveLength(2)
        expect(await screen.findByText("4")).toBeTruthy()
    })

    it("keeps the start of a running season and moves its end once confirmed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(2)).getByRole("button", {name: "Edit"}))
        expect((screen.getByLabelText("Starts") as HTMLInputElement).disabled).toBe(true)
        type("Ends", "2099-01-01T00:00")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText(/^Run season 2 from /)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.number).toBe(2)
        expect(asked.startsAt?.toDate()).toEqual(AT)
        expect(asked.endsAt?.toDate()).toEqual(new Date("2099-01-01T00:00"))
    })

    it("lets a season to come move its start", async () => {
        show(new FakeAdmin())

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(3)).getByRole("button", {name: "Edit"}))

        expect((screen.getByLabelText("Starts") as HTMLInputElement).disabled).toBe(false)
    })

    it("deletes the last season to come once confirmed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(3)).getByRole("button", {name: "Delete"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, delete"}))

        await waitFor(() => expect(rowOf(3)).toBeUndefined())
        expect(admin.askedFor("deleteSeason")).toEqual([{number: 3}])
    })

    it("quotes the server when it refuses a season", async () => {
        const admin = new FakeAdmin()
        admin.refuse("addSeason", Code.FailedPrecondition)
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        type("Starts", "2099-10-01T00:00")
        type("Ends", "2099-12-31T23:30")
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()
        expect(screen.getByRole("region", {name: "New season"})).toBeTruthy()
    })
})
