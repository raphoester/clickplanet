// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Report, ReportDecision, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {OpenPlayer} from "../navigation.ts"
import {ReportDetail} from "./ReportDetail.tsx"
import {Closed} from "./verdict.ts"

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

// Two more reports on Ana: one open, one closed.
const related = [
    new Report({id: "report-2", reportedAccountId: "account-reported", reason: ReportReason.BAD_NAME, createdAt: Timestamp.fromDate(AT)}),
    new Report({
        id: "report-3", reportedAccountId: "account-reported", reason: ReportReason.CHAT_ABUSE, createdAt: Timestamp.fromDate(AT),
        decision: new ReportDecision({outcome: ReportOutcome.DISMISSED}),
    }),
]

type Opened = {closed: Closed[], opened: string[]}

function open(admin: FakeAdmin, shown: Report = report, others: Report[] = []): Opened {
    const seen: Opened = {closed: [], opened: []}
    render(
        <ReportDetail
            report={shown}
            related={others}
            onBack={() => undefined}
            onOpen={id => seen.opened.push(id)}
            onClosed={closed => seen.closed.push(closed)}
            clients={admin.clients()}
        />,
    )
    return seen
}

// The lines of the conversation, once the pages around the reported message are read.
async function lines(): Promise<HTMLElement[]> {
    await screen.findByRole("list", {name: "The conversation"})
    await waitFor(() => expect(screen.queryByText("Loading what was said around it…")).toBeNull())
    return within(screen.getByRole("list", {name: "The conversation"})).getAllByRole("listitem")
}

// The chat around the reported message, shown once the message is read.
async function messagePanel() {
    return within(await screen.findByRole("region", {name: "The message"}))
}

function verdict() {
    return within(screen.getByRole("region", {name: "Verdict"}))
}

function confirm(label: string, yes: string) {
    fireEvent.click(verdict().getByRole("button", {name: label}))
    fireEvent.click(verdict().getByRole("button", {name: yes}))
}

describe("ReportDetail", () => {
    it("quotes the reported message, then reads a page of the chat on each side of it", async () => {
        const admin = new FakeAdmin()
        open(admin)

        const quote = await screen.findByText("you all suck", {selector: ".report-quote .line-text"})
        expect(quote).toBeTruthy()
        expect(screen.getByText("Bob wrote:")).toBeTruthy()
        await lines()
        expect(admin.askedFor("getMessage")).toEqual([{messageId: "message-1"}])
        expect(admin.askedFor("listMessages")).toEqual([
            {beforeMessageId: "message-1", limit: 10},
            {afterMessageId: "message-1", limit: 10},
        ])
    })

    it("shows the conversation oldest first, the reported message marked", async () => {
        open(new FakeAdmin())

        const shown = await lines()
        expect(shown.map(line => within(line).getByText(/^(come at me|calm down|you all suck|so rude)$/,
            {selector: ".line-text"}).textContent)).toEqual(["come at me", "calm down", "you all suck", "so rude"])
        expect(shown.map(line => line.getAttribute("aria-current"))).toEqual([null, null, "true", null])
        expect(within(shown[2]).getByText("Reported message")).toBeTruthy()
    })

    it("marks what the reporter and the reported account said, by account id", async () => {
        open(new FakeAdmin())

        const shown = await lines()
        expect(within(shown[0]).getByText("reporter")).toBeTruthy()
        expect(within(shown[1]).queryByText(/^report(er|ed)$/)).toBeNull()
        expect(within(shown[2]).getByText("reported")).toBeTruthy()
        expect(within(shown[3]).getByText("reporter")).toBeTruthy()
    })

    it("names each author and its country, and what a message answers", async () => {
        open(new FakeAdmin())

        const shown = await lines()
        expect(within(shown[1]).getByText("Cleo")).toBeTruthy()
        expect(shown[1].textContent).toContain("IT · ")
        expect(within(shown[1]).getByText(/^In reply to/).textContent).toBe("In reply to Bob: come at me")
        expect(within(shown[2]).getByText("Ana")).toBeTruthy()
    })

    it("pages back and forth from the reported message, on asking", async () => {
        const admin = new FakeAdmin()
        const said = (side: string, n: number) => new KeptMessage({
            messageId: `${side}-${n}`, accountId: "account-cleo", authorName: "Cleo", text: `${side} ${n}`,
            sentAt: Timestamp.fromDate(AT),
        })
        admin.chat = [
            ...Array.from({length: 25}, (_, n) => said("before", n + 1)),
            admin.chat[2],
            ...Array.from({length: 22}, (_, n) => said("after", n + 1)),
        ]
        open(admin)

        expect(await lines()).toHaveLength(21)
        const panel = await messagePanel()
        const list = screen.getByRole("list", {name: "The conversation"})
        expect(within(list).queryByText("before 15")).toBeNull()
        expect(within(list).getByText("before 16")).toBeTruthy()
        expect(within(list).getByText("after 10")).toBeTruthy()
        expect(within(list).queryByText("after 11")).toBeNull()

        fireEvent.click(panel.getByRole("button", {name: "Show earlier"}))
        expect(await within(list).findByText("before 1")).toBeTruthy()
        fireEvent.click(panel.getByRole("button", {name: "Show later"}))
        expect(await within(list).findByText("after 22")).toBeTruthy()

        expect(within(list).getAllByRole("listitem")).toHaveLength(48)
        expect(admin.askedFor("listMessages").slice(2)).toEqual([
            {beforeMessageId: "before-16", limit: 20},
            {afterMessageId: "after-10", limit: 20},
        ])
        expect(panel.queryByRole("button", {name: "Show earlier"})).toBeNull()
        expect(panel.queryByRole("button", {name: "Show later"})).toBeNull()
    })

    it("says when nothing was said around it", async () => {
        const admin = new FakeAdmin()
        admin.chat = [admin.chat[2]]
        open(admin)

        expect(await screen.findByText("Nothing said before it is kept.")).toBeTruthy()
        expect(screen.getByText("Nothing was said after it.")).toBeTruthy()
        expect((await messagePanel()).queryByRole("button", {name: /^Show/})).toBeNull()
    })

    it("strikes a message removed around it, with when", async () => {
        const admin = new FakeAdmin()
        admin.chat[1].removedAt = Timestamp.fromDate(AT)
        open(admin)

        const panel = await messagePanel()
        expect((await panel.findByText("calm down")).className).toContain("line-removed")
        expect(panel.getByText(/^Removed at /)).toBeTruthy()
        expect((verdict().getByRole("checkbox", {name: "Remove the message"}) as HTMLInputElement).disabled).toBe(false)
    })

    it("shows the reported message when the chat around it cannot be read, and reads it again", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listMessages", Code.Unavailable)
        open(admin)

        const panel = await messagePanel()
        expect(await panel.findByText("The call failed.")).toBeTruthy()
        expect(panel.getByText("you all suck")).toBeTruthy()
        admin.refusals.delete("listMessages")
        fireEvent.click(panel.getByRole("button", {name: "Try again"}))
        expect(await panel.findByText("so rude")).toBeTruthy()
    })

    it("says when an earlier page is no longer kept", async () => {
        const admin = new FakeAdmin()
        admin.chat = [
            ...Array.from({length: 10}, (_, n) => new KeptMessage({messageId: `before-${n}`, text: `before ${n}`})),
            admin.chat[2],
        ]
        open(admin)

        await lines()
        admin.refuse("listMessages", Code.NotFound)
        fireEvent.click((await messagePanel()).getByRole("button", {name: "Show earlier"}))
        expect(await screen.findByText("That part of the chat is no longer kept.")).toBeTruthy()
    })

    it("opens an author of the conversation, and either party, on the player page by its id", async () => {
        const openPlayer = vi.fn()
        render(
            <OpenPlayer.Provider value={openPlayer}>
                <ReportDetail report={report} onBack={() => undefined} clients={new FakeAdmin().clients()}/>
            </OpenPlayer.Provider>,
        )

        fireEvent.click(await screen.findByRole("button", {name: "Cleo"}))
        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: "account-cleo"}, name: "Cleo"})
        fireEvent.click(within(screen.getByRole("article")).getByRole("button", {name: "Ana"}))
        expect(openPlayer).toHaveBeenLastCalledWith({caller: {accountId: "account-reported"}, name: "Ana"})
    })

    it("says when the message is no longer kept, and offers no removal", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getMessage", Code.NotFound)
        open(admin)

        expect(await screen.findByText("This message is no longer kept.")).toBeTruthy()
        expect((verdict().getByRole("checkbox", {name: /^Remove the message/}) as HTMLInputElement).disabled).toBe(true)
    })

    it("says where the reported account stands with the antibot, and the whole reading on asking", async () => {
        const admin = new FakeAdmin()
        open(admin)

        expect(await screen.findByText("Not banned")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "What the antibot reads"}))
        expect(await screen.findByText(/^42 in /)).toBeTruthy()
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: "account-reported"}])
    })

    it("removes the message, mutes and closes the report as acted, with one question", async () => {
        const admin = new FakeAdmin()
        const seen = open(admin)

        await screen.findByText("you all suck", {selector: ".report-quote .line-text"})
        fireEvent.click(verdict().getByRole("checkbox", {name: "Remove the message"}))
        fireEvent.change(verdict().getByRole("combobox", {name: "Mute for"}), {target: {value: String(24 * 3600)}})
        fireEvent.click(verdict().getByRole("button", {name: "Act and close"}))
        expect(verdict().getByText("Remove the message, mute Ana for 1 day and close this report as Acted?")).toBeTruthy()
        fireEvent.click(verdict().getByRole("button", {name: "Yes, act and close"}))

        await waitFor(() => expect(seen.closed).toHaveLength(1))
        expect(admin.askedFor("removeMessage")).toEqual([{messageId: "message-1"}])
        const [muted] = admin.askedFor("mute") as {accountId: string, duration: {seconds: bigint}}[]
        expect(muted.accountId).toBe("account-reported")
        expect(muted.duration.seconds).toBe(86400n)
        expect(admin.askedFor("decideReport")).toEqual([
            {reportId: "report-1", outcome: ReportOutcome.ACTED, comment: "Removed the message. Muted for 1 day."},
        ])
        expect(seen.closed[0].said[0]).toBe("Removed the message.")
        expect(seen.closed[0].said.at(-1)).toBe("Closed 1 report as Acted.")
    })

    it("offers no removal of a message already removed", async () => {
        const admin = new FakeAdmin()
        admin.message.removedAt = Timestamp.fromDate(AT)
        open(admin)

        await screen.findByText(/^Removed /, {selector: ".report-quote .hint"})
        const remove = verdict().getByRole("checkbox", {name: "Remove the message (gone already)"}) as HTMLInputElement
        expect(remove.disabled).toBe(true)
    })

    it("bans the reported account on the ladder by default", async () => {
        const admin = new FakeAdmin()
        open(admin)

        fireEvent.click(verdict().getByRole("checkbox", {name: "Ban"}))
        confirm("Act and close", "Yes, act and close")
        await waitFor(() => expect(admin.askedFor("decideReport")).toHaveLength(1))
        expect(admin.askedFor("banPlayer")).toEqual([{accountId: "account-reported", duration: undefined}])
    })

    it("stops at a step the server refuses, and leaves the report open", async () => {
        const admin = new FakeAdmin()
        admin.refuse("banPlayer", Code.Unimplemented)
        const seen = open(admin)

        fireEvent.click(verdict().getByRole("checkbox", {name: "Mute"}))
        fireEvent.click(verdict().getByRole("checkbox", {name: "Ban"}))
        confirm("Act and close", "Yes, act and close")
        expect((await verdict().findByRole("alert")).textContent).toMatch(new RegExp(
            "^Muted until .*\\. The antibot is off on this server, so it cannot ban, unban or inspect\\. The report is still open\\.$"))
        expect(admin.askedFor("decideReport")).toEqual([])
        expect(seen.closed).toEqual([])
    })

    it("closes a report as acted with nothing done here, as after a ban from the player page", async () => {
        const admin = new FakeAdmin()
        open(admin)

        confirm("Close as acted", "Yes, close as acted")
        await waitFor(() => expect(admin.askedFor("decideReport")).toHaveLength(1))
        expect(admin.askedFor("decideReport")).toEqual([{reportId: "report-1", outcome: ReportOutcome.ACTED, comment: ""}])
    })

    it("dismisses the report with its comment", async () => {
        const admin = new FakeAdmin()
        const seen = open(admin)

        fireEvent.click(verdict().getByRole("checkbox", {name: "Mute"}))
        fireEvent.change(verdict().getByRole("textbox", {name: "Comment (optional)"}), {target: {value: "banter"}})
        fireEvent.click(verdict().getByRole("button", {name: "Dismiss"}))
        expect(verdict().getByText("Close this report as Dismissed?")).toBeTruthy()
        fireEvent.click(verdict().getByRole("button", {name: "Yes, dismiss"}))

        await waitFor(() => expect(seen.closed).toHaveLength(1))
        expect(admin.askedFor("mute")).toEqual([])
        expect(admin.askedFor("decideReport"))
            .toEqual([{reportId: "report-1", outcome: ReportOutcome.DISMISSED, comment: "banter"}])
        expect(seen.closed[0].outcome).toBe(ReportOutcome.DISMISSED)
    })

    it("closes the account's other open reports with it, unless asked not to", async () => {
        const admin = new FakeAdmin()
        open(admin, report, related)

        expect(screen.getByText("Reported 2 more times:")).toBeTruthy()
        const close = verdict().getByRole("checkbox", {name: "Close the other open report on Ana too"}) as HTMLInputElement
        expect(close.checked).toBe(true)
        fireEvent.click(verdict().getByRole("button", {name: "Dismiss"}))
        expect(verdict().getByText("Close these 2 reports as Dismissed?")).toBeTruthy()
        fireEvent.click(verdict().getByRole("button", {name: "Cancel"}))
        fireEvent.click(close)
        confirm("Dismiss", "Yes, dismiss")

        await waitFor(() => expect(admin.askedFor("decideReport")).toHaveLength(1))
        expect(admin.askedFor("decideReport"))
            .toEqual([{reportId: "report-1", outcome: ReportOutcome.DISMISSED, comment: ""}])
        cleanup()

        const again = new FakeAdmin()
        open(again, report, related)
        confirm("Dismiss", "Yes, dismiss")
        await waitFor(() => expect(again.askedFor("decideReport")).toHaveLength(2))
        expect(again.askedFor("decideReport").map(asked => (asked as {reportId: string}).reportId)).toEqual(["report-1", "report-2"])
    })

    it("opens the account's other reports from its own", async () => {
        const seen = open(new FakeAdmin(), report, related)

        fireEvent.click(screen.getByRole("button", {name: /^Bad name · .* · open$/}))
        expect(seen.opened).toEqual(["report-2"])
    })
})
