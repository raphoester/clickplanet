// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {Report, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {AT, FakeAdmin} from "./fakeClients.ts"
import {ReportDetail} from "./ReportDetail.tsx"

afterEach(cleanup)

const report = new Report({
    id: "report-1",
    reporterAccountId: "account-reporter",
    reportedAccountId: "account-reported",
    reporterName: "Bob",
    reportedName: "Ana",
    reason: ReportReason.CHAT_ABUSE,
    text: "insults",
    messageId: "message-1",
    createdAt: Timestamp.fromDate(AT),
})

function open(admin: FakeAdmin, shown: Report = report) {
    render(<ReportDetail report={shown} onBack={() => undefined} clients={admin.clients()}/>)
}

function confirm(label: string, yes: string) {
    fireEvent.click(screen.getByRole("button", {name: label}))
    fireEvent.click(screen.getByRole("button", {name: yes}))
}

function askedFor(admin: FakeAdmin, method: string) {
    return admin.asked.filter(call => call.method === method).map(call => call.request)
}

describe("ReportDetail", () => {
    it("reads the reported message", async () => {
        const admin = new FakeAdmin()
        open(admin)

        expect(await screen.findByText("you all suck")).toBeTruthy()
        expect(askedFor(admin, "getMessage")).toEqual([{messageId: "message-1"}])
    })

    it("says when the message is no longer kept", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getMessage", Code.NotFound)
        open(admin)

        expect(await screen.findByText("This message is no longer kept.")).toBeTruthy()
    })

    it("removes the message once confirmed, and shows it removed", async () => {
        const admin = new FakeAdmin()
        open(admin)
        await screen.findByText("you all suck")

        confirm("Remove the message", "Yes, remove the message")

        expect(await screen.findByText(/^Removed at /)).toBeTruthy()
        expect(askedFor(admin, "removeMessage")).toEqual([{messageId: "message-1"}])
        expect(screen.queryByRole("button", {name: "Remove the message"})).toBeNull()
    })

    it("offers no removal for a message already removed", async () => {
        const admin = new FakeAdmin()
        admin.message.removedAt = Timestamp.fromDate(AT)
        open(admin)

        expect(await screen.findByText(/^Removed at /)).toBeTruthy()
        expect(screen.queryByRole("button", {name: "Remove the message"})).toBeNull()
    })

    it("mutes the reported account for an hour by default", async () => {
        const admin = new FakeAdmin()
        open(admin)

        confirm("Mute", "Yes, mute")

        expect(await screen.findByText(/^Muted until /)).toBeTruthy()
        const [request] = askedFor(admin, "mute") as {accountId: string, duration: {seconds: bigint}}[]
        expect(request.accountId).toBe("account-reported")
        expect(request.duration.seconds).toBe(3600n)
    })

    it("bans the reported account on the ladder by default", async () => {
        const admin = new FakeAdmin()
        open(admin)

        fireEvent.click(screen.getByRole("button", {name: "Ban"}))
        expect(screen.getByText("Ban Ana for the ladder's next step?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, ban"}))

        expect(await screen.findByText(/^Banned until .*, offence 2\.$/)).toBeTruthy()
        expect(askedFor(admin, "banPlayer")).toEqual([{accountId: "account-reported", duration: undefined}])
    })

    it("says a ban the antibot cannot make in plain words", async () => {
        const admin = new FakeAdmin()
        admin.refuse("banPlayer", Code.FailedPrecondition)
        open(admin)

        confirm("Ban", "Yes, ban")

        expect(await screen.findByText("The antibot is off on this server, so it cannot ban or inspect.")).toBeTruthy()
    })

    it("inspects the reported account", async () => {
        const admin = new FakeAdmin()
        open(admin)

        fireEvent.click(screen.getByRole("button", {name: "Inspect"}))

        expect(await screen.findByText(/^42 in /)).toBeTruthy()
        expect(askedFor(admin, "inspectPlayer")).toEqual([{accountId: "account-reported"}])
    })

    it("closes the report with the outcome picked and its comment", async () => {
        const admin = new FakeAdmin()
        open(admin)

        expect((screen.getByRole("button", {name: "Close the report"}) as HTMLButtonElement).disabled).toBe(true)
        fireEvent.click(screen.getByRole("radio", {name: "Dismissed"}))
        fireEvent.change(screen.getByRole("textbox", {name: "Comment (optional)"}), {target: {value: "banter"}})
        confirm("Close the report", "Yes, close the report")

        expect(await screen.findByText(/^Closed as Dismissed at /)).toBeTruthy()
        expect(askedFor(admin, "decideReport"))
            .toEqual([{reportId: "report-1", outcome: ReportOutcome.DISMISSED, comment: "banter"}])
        const card = screen.getByRole("article")
        expect(within(card).getByText("Dismissed")).toBeTruthy()
        expect(within(card).getByText("banter")).toBeTruthy()
    })
})
