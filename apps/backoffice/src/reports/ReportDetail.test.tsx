// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Report, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {OpenPlayer} from "../navigation.ts"
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

describe("ReportDetail", () => {
    it("reads the reported message and the messages around it", async () => {
        const admin = new FakeAdmin()
        open(admin)

        expect(await screen.findByText("you all suck")).toBeTruthy()
        expect(admin.askedFor("getMessage")).toEqual([{messageId: "message-1", before: 50, after: 50}])
    })

    it("shows the conversation oldest first, the reported message marked", async () => {
        open(new FakeAdmin())

        const lines = within(await screen.findByRole("list", {name: "The conversation"})).getAllByRole("listitem")
        expect(lines.map(line => within(line).getByText(/^(come at me|calm down|you all suck|so rude)$/,
            {selector: ".line-text"}).textContent)).toEqual(["come at me", "calm down", "you all suck", "so rude"])
        expect(lines.map(line => line.getAttribute("aria-current"))).toEqual([null, null, "true", null])
        expect(within(lines[2]).getByText("Reported message")).toBeTruthy()
    })

    it("marks what the reporter and the reported account said, by account id", async () => {
        open(new FakeAdmin())

        const lines = within(await screen.findByRole("list", {name: "The conversation"})).getAllByRole("listitem")
        expect(within(lines[0]).getByText("reporter")).toBeTruthy()
        expect(within(lines[1]).queryByText(/^report(er|ed)$/)).toBeNull()
        expect(within(lines[2]).getByText("reported")).toBeTruthy()
        expect(within(lines[3]).getByText("reporter")).toBeTruthy()
    })

    it("names each author and its country, and what a message answers", async () => {
        open(new FakeAdmin())

        const lines = within(await screen.findByRole("list", {name: "The conversation"})).getAllByRole("listitem")
        expect(within(lines[1]).getByText("Cleo")).toBeTruthy()
        expect(lines[1].textContent).toContain("IT · ")
        expect(within(lines[2]).getByText(/^In reply to/).textContent).toBe("In reply to Bob: come at me")
    })

    it("shows the ten nearest on each side, and the rest on asking", async () => {
        const admin = new FakeAdmin()
        const said = (side: string, n: number) => new KeptMessage({
            messageId: `${side}-${n}`, accountId: "account-cleo", authorName: "Cleo", text: `${side} ${n}`,
            sentAt: Timestamp.fromDate(AT),
        })
        admin.message.before = Array.from({length: 14}, (_, n) => said("before", n + 1))
        admin.message.after = Array.from({length: 12}, (_, n) => said("after", n + 1))
        open(admin)

        const list = await screen.findByRole("list", {name: "The conversation"})
        expect(within(list).getAllByRole("listitem")).toHaveLength(21)
        expect(within(list).queryByText("before 4")).toBeNull()
        expect(within(list).getByText("before 5")).toBeTruthy()
        expect(within(list).getByText("after 10")).toBeTruthy()
        expect(within(list).queryByText("after 11")).toBeNull()

        fireEvent.click(screen.getByRole("button", {name: "Show 4 earlier"}))
        fireEvent.click(screen.getByRole("button", {name: "Show 2 later"}))

        expect(within(list).getAllByRole("listitem")).toHaveLength(27)
        expect(within(list).getByText("before 1")).toBeTruthy()
        expect(within(list).getByText("after 12")).toBeTruthy()
    })

    it("says when nothing was said around it", async () => {
        const admin = new FakeAdmin()
        admin.message.before = []
        admin.message.after = []
        open(admin)

        expect(await screen.findByText("Nothing said before it is kept.")).toBeTruthy()
        expect(screen.getByText("Nothing was said after it.")).toBeTruthy()
    })

    it("strikes a message removed around it, with when", async () => {
        const admin = new FakeAdmin()
        admin.message.before[1].removedAt = Timestamp.fromDate(AT)
        open(admin)

        expect((await screen.findByText("calm down")).className).toContain("line-removed")
        expect(screen.getByText(/^Removed at /)).toBeTruthy()
        expect(screen.getByRole("button", {name: "Remove the message"})).toBeTruthy()
    })

    it("opens an author of the conversation on the player page, by its id", async () => {
        const openPlayer = vi.fn()
        render(
            <OpenPlayer.Provider value={openPlayer}>
                <ReportDetail report={report} onBack={() => undefined} clients={new FakeAdmin().clients()}/>
            </OpenPlayer.Provider>,
        )

        fireEvent.click(await screen.findByRole("button", {name: "Cleo"}))

        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: "account-cleo"}, name: "Cleo"})
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
        expect(admin.askedFor("removeMessage")).toEqual([{messageId: "message-1"}])
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
        const [request] = admin.askedFor("mute") as {accountId: string, duration: {seconds: bigint}}[]
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
        expect(admin.askedFor("banPlayer")).toEqual([{accountId: "account-reported", duration: undefined}])
    })

    it("says a ban the antibot cannot make in plain words", async () => {
        const admin = new FakeAdmin()
        admin.refuse("banPlayer", Code.FailedPrecondition)
        open(admin)

        confirm("Ban", "Yes, ban")

        expect(await screen.findByText("The antibot is off on this server, so it cannot ban, unban or inspect.")).toBeTruthy()
    })

    it("inspects the reported account", async () => {
        const admin = new FakeAdmin()
        open(admin)

        fireEvent.click(screen.getByRole("button", {name: "Inspect"}))

        expect(await screen.findByText(/^42 in /)).toBeTruthy()
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: "account-reported"}])
    })

    it("opens the reported account on the player page, by its id", () => {
        const openPlayer = vi.fn()
        render(
            <OpenPlayer.Provider value={openPlayer}>
                <ReportDetail report={report} onBack={() => undefined} clients={new FakeAdmin().clients()}/>
            </OpenPlayer.Provider>,
        )

        fireEvent.click(screen.getByRole("button", {name: "Every tool →"}))

        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: "account-reported"}, name: "Ana"})
    })

    it("closes the report with the outcome picked and its comment", async () => {
        const admin = new FakeAdmin()
        open(admin)

        expect((screen.getByRole("button", {name: "Close the report"}) as HTMLButtonElement).disabled).toBe(true)
        fireEvent.click(screen.getByRole("radio", {name: "Dismissed"}))
        fireEvent.change(screen.getByRole("textbox", {name: "Comment (optional)"}), {target: {value: "banter"}})
        confirm("Close the report", "Yes, close the report")

        expect(await screen.findByText(/^Closed as Dismissed at /)).toBeTruthy()
        expect(admin.askedFor("decideReport"))
            .toEqual([{reportId: "report-1", outcome: ReportOutcome.DISMISSED, comment: "banter"}])
        const card = screen.getByRole("article")
        expect(within(card).getByText("Dismissed")).toBeTruthy()
        expect(within(card).getByText("banter")).toBeTruthy()
    })
})
