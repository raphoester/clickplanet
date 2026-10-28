// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {Report, ReportDecision, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {MAX_REPORTS, ReportsView} from "./ReportsView.tsx"

afterEach(cleanup)

// Ana twice and Dan once, open; Eve's, closed.
function withReports(): FakeAdmin {
    const admin = new FakeAdmin()
    const report = (id: string, reported: string, name: string, part: Partial<Report> = {}) => new Report({
        id, reportedAccountId: reported, reportedName: name, reporterAccountId: "account-reporter", reporterName: "Bob",
        reason: ReportReason.CHAT_ABUSE, createdAt: Timestamp.fromDate(AT), ...part,
    })
    admin.reports = [
        report("report-1", "account-reported", "Ana", {messageId: "message-1"}),
        report("report-2", "account-dan", "Dan", {reason: ReportReason.BAD_NAME, text: "rude name"}),
        report("report-3", "account-reported", "Ana", {reason: ReportReason.OTHER, text: "again"}),
        report("report-4", "account-eve", "Eve", {
            text: "spam", decision: new ReportDecision({outcome: ReportOutcome.ACTED, comment: "Muted for 1 day."}),
        }),
    ]
    return admin
}

function Inbox({admin, first = ""}: {admin: FakeAdmin, first?: string}) {
    const [opened, setOpened] = useState(first)
    return (
        <>
            <p data-testid="opened">{opened}</p>
            <ReportsView opened={opened} onOpen={setOpened} clients={admin.clients()}/>
        </>
    )
}

function list() {
    return within(screen.getByRole("list", {name: "The reports"}))
}

describe("the reports inbox", () => {
    it("lists the open reports with what was said, each account's count, and reads every report once", async () => {
        const admin = withReports()
        render(<Inbox admin={admin}/>)

        expect(await screen.findByRole("button", {name: "Open (3)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "All (4)"})).toBeTruthy()
        const rows = list().getAllByRole("listitem")
        expect(rows).toHaveLength(3)
        expect(await within(rows[0]).findByText("you all suck")).toBeTruthy()
        expect(within(rows[0]).getByText("×2")).toBeTruthy()
        expect(within(rows[1]).getByText("rude name")).toBeTruthy()
        expect(within(rows[1]).queryByText(/^×/)).toBeNull()
        expect(admin.askedFor("listReports")).toEqual([{limit: MAX_REPORTS}])
        expect(screen.getByText("Pick a report on the left.")).toBeTruthy()
    })

    it("shows the closed ones too, with what was decided", async () => {
        render(<Inbox admin={withReports()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "All (4)"}))
        const eve = list().getAllByRole("listitem")[3]
        expect(within(eve).getByText("Acted")).toBeTruthy()
        expect(eve.textContent).toContain(": Muted for 1 day.")
    })

    it("opens a report picked, then the next open one once a verdict closes it, and says what it did", async () => {
        const admin = withReports()
        render(<Inbox admin={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: /Dan/}))
        expect(screen.getByTestId("opened").textContent).toBe("report-2")
        const verdict = within(await screen.findByRole("region", {name: "Verdict"}))
        fireEvent.click(verdict.getByRole("button", {name: "Dismiss"}))
        fireEvent.click(verdict.getByRole("button", {name: "Yes, dismiss"}))

        expect((await screen.findByRole("status")).textContent).toBe("Dan: Closed 1 report as Dismissed.")
        await waitFor(() => expect(screen.getByTestId("opened").textContent).toBe("report-3"))
        expect(screen.getByRole("button", {name: "Open (2)"})).toBeTruthy()
        expect(list().queryByText("Dan")).toBeNull()
    })

    it("closes an account's reports together, and opens the first one left", async () => {
        const admin = withReports()
        render(<Inbox admin={admin} first="report-3"/>)

        const verdict = within(await screen.findByRole("region", {name: "Verdict"}))
        fireEvent.click(verdict.getByRole("checkbox", {name: "Mute"}))
        fireEvent.click(verdict.getByRole("button", {name: "Act and close"}))
        fireEvent.click(verdict.getByRole("button", {name: "Yes, act and close"}))

        await waitFor(() => expect(screen.getByTestId("opened").textContent).toBe("report-2"))
        expect(admin.askedFor("decideReport").map(asked => (asked as {reportId: string}).reportId)).toEqual(["report-3", "report-1"])
        expect(screen.getByRole("button", {name: "Open (1)"})).toBeTruthy()
    })

    it("says when nothing is left to judge", async () => {
        const admin = new FakeAdmin()
        admin.reports = []
        render(<Inbox admin={admin}/>)

        expect(await screen.findByText("No open report. All done.")).toBeTruthy()
    })
})
