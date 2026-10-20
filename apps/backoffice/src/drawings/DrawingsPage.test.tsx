// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {ReportDecision, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {OpenPlayer} from "../navigation.ts"
import {localInputOf, OLDER_THAN_THE_LEDGER} from "./actions.ts"
import {DrawingsView} from "./DrawingsView.tsx"

afterEach(cleanup)

function show(admin: FakeAdmin) {
    render(<DrawingsView clients={admin.clients()}/>)
}

// The card's Open, not the switch to the open reports.
async function openCard() {
    await screen.findByText("a swastika over Berlin")
    fireEvent.click(within(screen.getByRole("listitem")).getByRole("button", {name: "Open"}))
}

async function openTheReport(admin: FakeAdmin) {
    show(admin)
    await openCard()
}

function region(name: string) {
    return within(screen.getByRole("region", {name}))
}

function button(panel: string, name: string): HTMLButtonElement {
    return region(panel).getByRole("button", {name}) as HTMLButtonElement
}

describe("Drawings", () => {
    it("lists the open reports first, and all on asking", async () => {
        const admin = new FakeAdmin()
        show(admin)

        expect(await screen.findByText("a swastika over Berlin")).toBeTruthy()
        expect(screen.getByText("Hate")).toBeTruthy()
        expect(screen.getByText("Bob")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "All"}))
        await screen.findByText("1 report")

        expect(admin.askedFor("listPlaceReports")).toEqual([{openOnly: true}, {openOnly: false}])
    })

    it("shows the decision of a report decided", async () => {
        const admin = new FakeAdmin()
        admin.placeReports[0].decision = new ReportDecision({
            outcome: ReportOutcome.DISMISSED, comment: "a flag, not a symbol", decidedAt: Timestamp.fromDate(AT),
        })
        show(admin)
        fireEvent.click(await screen.findByRole("button", {name: "All"}))

        expect(await screen.findByText("Dismissed")).toBeTruthy()
        expect(screen.getByText("a flag, not a symbol")).toBeTruthy()
    })

    it("links the drawing on the map, in a new tab", async () => {
        await openTheReport(new FakeAdmin())

        const link = screen.getByRole("link", {name: "See it on the map ↗"})
        expect(link.getAttribute("href")).toBe("https://clickplanet.lol/?tile=1234")
        expect(link.getAttribute("target")).toBe("_blank")
    })

    it("restores only after a dry run, asking with its numbers", async () => {
        const admin = new FakeAdmin()
        await openTheReport(admin)

        expect(button("Restore", "Restore").disabled).toBe(true)
        fireEvent.click(button("Restore", "Dry run"))

        expect(await region("Restore").findByText(/^37 tiles in the area, 12 changed since /)).toBeTruthy()
        expect(admin.askedFor("restoreArea")).toEqual([{
            tileId: 1234, rings: 8, before: Timestamp.fromDate(new Date(AT.getTime() - 3_600_000)), dryRun: true,
        }])
        fireEvent.click(button("Restore", "Restore"))
        expect(region("Restore").getByText(/^Put back the 12 tiles changed since /)).toBeTruthy()
        fireEvent.click(button("Restore", "Yes, restore"))

        expect(await region("Restore").findByText("Restored 12 of 12 tiles.")).toBeTruthy()
        expect(admin.askedFor("restoreArea").at(-1)).toMatchObject({tileId: 1234, rings: 8, dryRun: false})
    })

    it("needs a new dry run once the area changes", async () => {
        const admin = new FakeAdmin()
        await openTheReport(admin)
        fireEvent.click(button("Restore", "Dry run"))
        await region("Restore").findByText(/^37 tiles in the area/)

        fireEvent.change(region("The area").getByRole("spinbutton"), {target: {value: "12"}})

        expect(button("Restore", "Restore").disabled).toBe(true)
        expect(region("Restore").queryByText(/^37 tiles in the area/)).toBeNull()
    })

    it("names the painters, each opening on the player page", async () => {
        const admin = new FakeAdmin()
        const openPlayer = vi.fn()
        render(
            <OpenPlayer.Provider value={openPlayer}>
                <DrawingsView clients={admin.clients()}/>
            </OpenPlayer.Provider>,
        )
        await openCard()
        fireEvent.click(button("Restore", "Dry run"))

        const painters = await region("Restore").findByRole("list", {name: "Painters"})
        expect(within(painters).getByText("BraveFox42")).toBeTruthy()
        expect(within(painters).getByText("No account")).toBeTruthy()
        expect(within(painters).getByText("40 takes, 12 held")).toBeTruthy()
        fireEvent.click(within(painters).getAllByRole("button", {name: "Open"})[0])

        expect(openPlayer).toHaveBeenCalledWith(expect.objectContaining({caller: {accountId: ACCOUNT}, name: "BraveFox42"}))
    })

    it("says when the ledger does not reach back far enough, and the clear is there", async () => {
        const admin = new FakeAdmin()
        await openTheReport(admin)

        fireEvent.change(region("Restore").getByLabelText("As it was at"),
            {target: {value: localInputOf(new Date(admin.keptSince.getTime() - 3_600_000))}})
        fireEvent.click(button("Restore", "Dry run"))

        expect(await region("Restore").findByText(OLDER_THAN_THE_LEDGER)).toBeTruthy()
        expect(button("Restore", "Restore").disabled).toBe(true)
        expect(button("Clear", "Dry run").disabled).toBe(false)
    })

    it("clears the area only after a dry run, asking with its numbers", async () => {
        const admin = new FakeAdmin()
        await openTheReport(admin)

        expect(button("Clear", "Clear").disabled).toBe(true)
        fireEvent.click(button("Clear", "Dry run"))
        expect(await region("Clear").findByText("37 tiles in the area, 20 held by a flag.")).toBeTruthy()
        fireEvent.click(button("Clear", "Clear"))
        expect(region("Clear").getByText("Give the 20 tiles held in the area to nobody?")).toBeTruthy()
        fireEvent.click(button("Clear", "Yes, clear"))

        expect(await region("Clear").findByText("Cleared 20 of 20 tiles held.")).toBeTruthy()
        expect(admin.askedFor("clearArea")).toEqual([
            {tileId: 1234, rings: 8, dryRun: true},
            {tileId: 1234, rings: 8, dryRun: false},
        ])
    })

    it("closes the report with the outcome picked and its comment", async () => {
        const admin = new FakeAdmin()
        await openTheReport(admin)

        fireEvent.click(screen.getByRole("radio", {name: "Acted"}))
        fireEvent.change(screen.getByRole("textbox", {name: "Comment (optional)"}), {target: {value: "restored"}})
        fireEvent.click(screen.getByRole("button", {name: "Close the report"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, close the report"}))

        expect(await screen.findByText(/^Closed as Acted at /)).toBeTruthy()
        expect(admin.askedFor("decidePlaceReport"))
            .toEqual([{reportId: "place-report-1", outcome: ReportOutcome.ACTED, comment: "restored"}])
    })
})
