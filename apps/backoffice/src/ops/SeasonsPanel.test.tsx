// @vitest-environment jsdom
import {Duration} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {clockOf, localOf, timeOf} from "../components/time.ts"
import {FakeAdmin} from "../fakeAdmin.ts"
import {AddSeasonRequest, EditSeasonRequest} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {SeasonsPanel} from "./SeasonsPanel.tsx"

afterEach(cleanup)

const DAY = 24 * 3600
const HOUR = 3600

const rowOf = (number: number) =>
    screen.getAllByRole("row").find(row => row.querySelector("strong")?.textContent === String(number)) as HTMLElement

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const field = (label: string) => screen.getByLabelText(label) as HTMLInputElement

const startOf = (admin: FakeAdmin, number: number) => admin.seasons.seasons[number].season?.startsAt?.toDate() as Date
const endOf = (admin: FakeAdmin, number: number) => admin.seasons.seasons[number].season?.endsAt?.toDate() as Date
const later = (date: Date, seconds: number) => new Date(date.getTime() + seconds * 1000)

const alerts = () => screen.queryAllByRole("alert").map(alert => alert.textContent)

function show(admin: FakeAdmin) {
    render(<SeasonsPanel gamemaster={admin.gamemaster} planet={admin} antibot={admin.antibot}/>)
}

async function edit(number: number) {
    await screen.findByRole("table")
    fireEvent.click(within(rowOf(number)).getByRole("button", {name: "Edit"}))
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

    it("starts a new season as the last one ends, for 28 days, in the operator's own time", async () => {
        const admin = new FakeAdmin()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        expect(field("Starts").value).toBe(localOf(endOf(admin, 2)))
        expect(field("Ends").value).toBe(localOf(later(endOf(admin, 2), 28 * DAY)))
        expect(screen.getByText("28 d long, starting as season 2 ends.")).toBeTruthy()
        expect(screen.getByText(`Times are in ${Intl.DateTimeFormat().resolvedOptions().timeZone}.`)).toBeTruthy()
    })

    it("adds a season from the times picked, as the break and the length they give, once confirmed", async () => {
        const admin = new FakeAdmin()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        const startsAt = later(endOf(admin, 2), 16 * HOUR)
        const endsAt = later(startsAt, 30 * DAY)
        type("Starts", localOf(startsAt))
        type("Ends", localOf(endsAt))
        expect(screen.getByText("30 d long, starting 16 h after season 2 ends.")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        expect(screen.getByText(`Add season 3 from ${timeOf(startsAt)} to ${timeOf(endsAt)} (30 d long, starting 16 h after season 2 ends), with no final battle?`))
            .toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        await waitFor(() => expect(screen.queryByRole("region", {name: "New season"})).toBeNull())
        const [asked] = admin.askedFor("addSeason") as AddSeasonRequest[]
        expect(asked.breakBefore?.seconds).toBe(BigInt(16 * HOUR))
        expect(asked.length?.seconds).toBe(BigInt(30 * DAY))
        expect(admin.askedFor("getSeasons")).toHaveLength(2)
        expect(await screen.findByText("3")).toBeTruthy()
    })

    it("keeps the start of a running season and moves its end once confirmed, and the seasons after it", async () => {
        const admin = new FakeAdmin()
        const before = endOf(admin, 2)
        show(admin)

        await edit(1)
        expect(field("Starts").disabled).toBe(true)
        const endsAt = new Date("2099-01-01T21:00:00Z")
        type("Ends", localOf(endsAt))
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText(/^Run season 1 from /)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.number).toBe(1)
        expect(asked.breakBefore?.seconds).toBe(BigInt(DAY))
        expect(asked.length?.seconds).toBe(BigInt((endsAt.getTime() - startOf(admin, 1).getTime()) / 1000))
        expect(endOf(admin, 2)).not.toEqual(before)
    })

    it("sends back a duration left as it was exactly as the server kept it", async () => {
        const admin = new FakeAdmin()
        const kept = new Duration({seconds: BigInt(90 * DAY + 1234), nanos: 5})
        admin.seasons.seasons[2].length = kept
        show(admin)

        await edit(2)
        type("Description", "Four times the points.")
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.length?.equals(kept)).toBe(true)
        expect(asked.breakBefore?.seconds).toBe(0n)
    })

    it("moves the end alone when only the start moves", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await edit(2)
        type("Starts", localOf(later(startOf(admin, 2), DAY)))
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.breakBefore?.seconds).toBe(BigInt(DAY))
        expect(asked.length?.seconds).toBe(BigInt(89 * DAY))
    })

    it("says when days end, in the operator's own time", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await edit(2)
        const cutoff = new Date(Date.UTC(endOf(admin, 2).getUTCFullYear(), endOf(admin, 2).getUTCMonth(), endOf(admin, 2).getUTCDate(), 21))
        expect(screen.getByText(`Days end at ${clockOf(cutoff)}.`)).toBeTruthy()
    })

    it("shows each season's final battle with its times, and keeps it on an edit of the rest", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await screen.findByRole("table")
        expect(within(rowOf(2)).getByText(/2 h, rush × 3/)).toBeTruthy()
        expect(within(rowOf(1)).getByText("none")).toBeTruthy()
        fireEvent.click(within(rowOf(2)).getByRole("button", {name: "Edit"}))
        expect(field("Ends with a final battle").checked).toBe(true)
        expect(field("Final battle, hours").value).toBe("2")
        const shown = Array.from(screen.getByText(/^Final battle from/).querySelectorAll("time")).map(time => time.getAttribute("dateTime"))
        expect(shown).toEqual([later(endOf(admin, 2), -2 * HOUR).toISOString(), endOf(admin, 2).toISOString()])
        type("Starts", localOf(later(startOf(admin, 2), DAY)))
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText(/ending with a final battle of 2 h, rush × 3 from .+\?$/)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("editSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("editSeason") as EditSeasonRequest[]
        expect(asked.finale?.ruleSet).toBe("rush")
        expect(asked.finale?.times).toBe(3)
        expect(asked.finale?.description).toBe("Triple points.")
        expect(asked.finale?.length).toBe(admin.seasons.seasons[2].finale?.length)
    })

    it("warns of a final battle that neither starts nor ends when a day ends, and of nothing once it ends then", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await edit(2)
        const end = endOf(admin, 2)
        const cutoff = new Date(Date.UTC(end.getUTCFullYear(), end.getUTCMonth(), end.getUTCDate(), 21))
        expect(alerts()).toEqual([
            `The final battle runs ${clockOf(later(end, -2 * HOUR))} to ${clockOf(end)}: it must start or end at ${clockOf(cutoff)}, when a day ends.`,
        ])
        type("Ends", localOf(cutoff))
        expect(alerts()).toEqual([])
    })

    it("warns of a season that starts before the one before ends, or ends before it starts", async () => {
        const admin = new FakeAdmin()
        show(admin)

        await edit(2)
        type("Starts", localOf(later(endOf(admin, 1), -HOUR)))
        expect(alerts()).toContain(`It starts before season 1 ends, at ${timeOf(endOf(admin, 1))}.`)
        type("Ends", localOf(later(endOf(admin, 1), -2 * HOUR)))
        expect(alerts()).toContain("It ends before it starts.")
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(false)
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

    it("adds a season with a final battle once its set, among the ones every module keeps, and its times are picked", async () => {
        const admin = new FakeAdmin()
        show(admin)

        fireEvent.click(await screen.findByRole("button", {name: "Add a season"}))
        fireEvent.click(screen.getByLabelText("Ends with a final battle"))
        expect((screen.getByRole("button", {name: "Add"}) as HTMLButtonElement).disabled).toBe(true)
        expect(within(screen.getByLabelText("Rule set")).getAllByRole("option").map(option => option.textContent))
            .toEqual(["Pick a set", "default"])
        type("Rule set", "default")
        type("Final battle, hours", "4")
        type("Times", "3")
        type("Description", "Triple points.")
        fireEvent.click(screen.getByRole("button", {name: "Add"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, add"}))

        await waitFor(() => expect(admin.askedFor("addSeason")).toHaveLength(1))
        const [asked] = admin.askedFor("addSeason") as AddSeasonRequest[]
        expect(asked.finale?.length?.seconds).toBe(BigInt(4 * HOUR))
        expect(asked.finale?.ruleSet).toBe("default")
        expect(asked.finale?.times).toBe(3)
    })
})
