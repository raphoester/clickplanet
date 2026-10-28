// @vitest-environment jsdom
import {Duration} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {AddSeasonRequest, EditSeasonRequest} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {SeasonsPanel} from "./SeasonsPanel.tsx"

afterEach(cleanup)

const DAY = 24 * 3600

const rowOf = (number: number) =>
    screen.getAllByRole("row").find(row => row.querySelector("strong")?.textContent === String(number)) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const endOf = (admin: FakeAdmin, number: number) => admin.seasons.seasons[number].season?.endsAt?.toDate() as Date

// The window the form shows, as the times' dateTime.
const windowShown = () =>
    Array.from(within(screen.getByRole("region", {name: /season/})).getByText(/^From/).querySelectorAll("time")).map(time => time.getAttribute("dateTime"))

function show(admin: FakeAdmin) {
    render(<SeasonsPanel gamemaster={admin.gamemaster}/>)
}

describe("the seasons page", () => {
    it("lists every season by number with its break and length, the running one marked, the done one muted", async () => {
        show(new FakeAdmin())

        await screen.findByRole("table")
        const numbers = screen.getAllByRole("row").slice(1).map(row => row.querySelector("strong")?.textContent)
        expect(numbers).toEqual(["0", "1", "2"])
        expect(within(rowOf(0)).getByText("done")).toBeTruthy()
        expect(rowOf(0).className).toBe("muted")
        expect(within(rowOf(1)).getByText("running")).toBeTruthy()
        expect(within(rowOf(1)).getByText("1 d")).toBeTruthy()
        expect(rowOf(1).getAttribute("aria-current")).toBe("true")
        expect(within(rowOf(2)).getByText("to come")).toBeTruthy()
        expect(within(rowOf(2)).getByText("0 h")).toBeTruthy()
        expect(within(rowOf(2)).getByText("90 d")).toBeTruthy()
    })

    it("offers an edit of a season that has not ended, and a delete of the last one to come", async () => {
        show(new FakeAdmin())

        await screen.findByRole("table")
        expect(within(rowOf(0)).queryByRole("button")).toBeNull()
        expect(within(rowOf(1)).getByRole("button", {name: "Edit"})).toBeTruthy()
        expect(within(rowOf(1)).queryByRole("button", {name: "Delete"})).toBeNull()
        expect(within(rowOf(2)).getByRole("button", {name: "Edit"})).toBeTruthy()
        expect(within(rowOf(2)).getByRole("button", {name: "Delete"})).toBeTruthy()
    })

    it("adds a season of 28 days after the last one by default, its window shown, once confirmed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        const startsAt = endOf(admin, 2)
        expect(windowShown()).toEqual([startsAt.toISOString(), new Date(startsAt.getTime() + 28 * DAY * 1000).toISOString()])
        type("Length, days", "30")
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        expect(screen.getByText(/^Add a season: a break of 0 h, then 30 d, from /)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        await waitFor(() => expect(screen.queryByRole("region", {name: "New season"})).toBeNull())
        const [asked] = admin.askedFor("addSeason") as AddSeasonRequest[]
        expect(asked.breakBefore?.seconds).toBe(0n)
        expect(asked.length?.seconds).toBe(BigInt(30 * DAY))
        expect(admin.askedFor("getSeasons")).toHaveLength(2)
        expect(await screen.findByText("3")).toBeTruthy()
    })

    it("shows the window the durations give as they are typed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(2)).getByRole("button", {name: "Edit"}))
        type("Break, days", "2")
        type("Break, hours", "6")
        type("Length, days", "10")

        const startsAt = new Date(endOf(admin, 1).getTime() + (2 * DAY + 6 * 3600) * 1000)
        expect(windowShown()).toEqual([startsAt.toISOString(), new Date(startsAt.getTime() + 10 * DAY * 1000).toISOString()])
    })

    it("keeps the break of a running season and moves its length once confirmed, and the seasons after it", async () => {
        const admin = new FakeAdmin()
        const before = endOf(admin, 2)
        show(admin)

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(1)).getByRole("button", {name: "Edit"}))
        expect((screen.getByLabelText("Break, days") as HTMLInputElement).disabled).toBe(true)
        expect((screen.getByLabelText("Break, hours") as HTMLInputElement).disabled).toBe(true)
        type("Length, days", "40000")
        type("Length, hours", "0")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText(/^Run season 1: /)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.number).toBe(1)
        expect(asked.breakBefore?.seconds).toBe(BigInt(DAY))
        expect(asked.length?.seconds).toBe(BigInt(40000 * DAY))
        expect(endOf(admin, 2)).not.toEqual(before)
    })

    it("sends back a duration left as it was exactly as the server kept it", async () => {
        const admin = new FakeAdmin()
        const kept = new Duration({seconds: BigInt(90 * DAY + 1234), nanos: 5})
        admin.seasons.seasons[2].length = kept
        show(admin)

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(2)).getByRole("button", {name: "Edit"}))
        expect((screen.getByLabelText("Break, days") as HTMLInputElement).disabled).toBe(false)
        type("Break, days", "1")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.breakBefore?.seconds).toBe(BigInt(DAY))
        expect(asked.length?.equals(kept)).toBe(true)
    })

    it("deletes the last season to come once confirmed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await screen.findByRole("table")
        fireEvent.click(within(rowOf(2)).getByRole("button", {name: "Delete"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, delete"}))

        await waitFor(() => expect(rowOf(2)).toBeUndefined())
        expect(admin.askedFor("deleteSeason")).toEqual([{number: 2}])
    })

    it("quotes the server when it refuses a season", async () => {
        const admin = new FakeAdmin()
        admin.refuse("addSeason", Code.FailedPrecondition)
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()
        expect(screen.getByRole("region", {name: "New season"})).toBeTruthy()
    })
})
