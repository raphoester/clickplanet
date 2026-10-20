// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
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

// The lines of the conversation, once the pages around the reported message are read.
async function lines(): Promise<HTMLElement[]> {
    await screen.findByRole("list", {name: "The conversation"})
    await waitFor(() => expect(screen.queryByText("Loading what was said around it…")).toBeNull())
    return within(screen.getByRole("list", {name: "The conversation"})).getAllByRole("listitem")
}

// The reported message's own panel: the page also lists every message of the reported account.
function messagePanel() {
    return within(screen.getByRole("region", {name: "The message"}))
}

// What the conversation around the reported message asked, leaving out the reported account's own messages.
function asksAround(admin: FakeAdmin): unknown[] {
    return admin.askedFor("listMessages").filter(ask => !(ask as {accountId?: string}).accountId)
}

function confirm(label: string, yes: string) {
    fireEvent.click(screen.getByRole("button", {name: label}))
    fireEvent.click(screen.getByRole("button", {name: yes}))
}

describe("ReportDetail", () => {
    it("reads the reported message, then a page of the chat on each side of it", async () => {
        const admin = new FakeAdmin()
        open(admin)

        expect(await messagePanel().findByText("you all suck")).toBeTruthy()
        await lines()
        expect(admin.askedFor("getMessage")).toEqual([{messageId: "message-1"}])
        expect(asksAround(admin)).toEqual([
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
        const list = screen.getByRole("list", {name: "The conversation"})
        expect(within(list).queryByText("before 15")).toBeNull()
        expect(within(list).getByText("before 16")).toBeTruthy()
        expect(within(list).getByText("after 10")).toBeTruthy()
        expect(within(list).queryByText("after 11")).toBeNull()

        fireEvent.click(screen.getByRole("button", {name: "Show earlier"}))
        expect(await within(list).findByText("before 1")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Show later"}))
        expect(await within(list).findByText("after 22")).toBeTruthy()

        expect(within(list).getAllByRole("listitem")).toHaveLength(48)
        expect(asksAround(admin).slice(2)).toEqual([
            {beforeMessageId: "before-16", limit: 20},
            {afterMessageId: "after-10", limit: 20},
        ])
        expect(screen.queryByRole("button", {name: "Show earlier"})).toBeNull()
        expect(screen.queryByRole("button", {name: "Show later"})).toBeNull()
    })

    it("says when nothing was said around it", async () => {
        const admin = new FakeAdmin()
        admin.chat = [admin.chat[2]]
        open(admin)

        expect(await screen.findByText("Nothing said before it is kept.")).toBeTruthy()
        expect(screen.getByText("Nothing was said after it.")).toBeTruthy()
        expect(messagePanel().queryByRole("button", {name: /^Show/})).toBeNull()
    })

    it("strikes a message removed around it, with when", async () => {
        const admin = new FakeAdmin()
        admin.chat[1].removedAt = Timestamp.fromDate(AT)
        open(admin)

        expect((await screen.findByText("calm down")).className).toContain("line-removed")
        expect(screen.getByText(/^Removed at /)).toBeTruthy()
        expect(screen.getByRole("button", {name: "Remove the message"})).toBeTruthy()
    })

    it("shows the reported message when the chat around it cannot be read, and reads it again", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listMessages", Code.Unavailable)
        open(admin)

        expect(await messagePanel().findByText("The call failed.")).toBeTruthy()
        expect(messagePanel().getByText("you all suck")).toBeTruthy()

        admin.refusals.delete("listMessages")
        fireEvent.click(messagePanel().getByRole("button", {name: "Try again"}))

        expect(await messagePanel().findByText("so rude")).toBeTruthy()
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
        fireEvent.click(screen.getByRole("button", {name: "Show earlier"}))

        expect(await screen.findByText("That part of the chat is no longer kept.")).toBeTruthy()
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
        await messagePanel().findByText("you all suck")

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
