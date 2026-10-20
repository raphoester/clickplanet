// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Report} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {ReportDetail} from "./ReportDetail.tsx"

afterEach(cleanup)

// A report from the player's card: it names the account, and no message.
const fromTheCard = new Report({
    id: "report-2",
    reporterAccountId: "account-reporter",
    reportedAccountId: "account-reported",
    reporterName: "Bob",
    reportedName: "Ana",
    reason: ReportReason.CHAT_ABUSE,
    createdAt: Timestamp.fromDate(AT),
})

function said(n: number, accountId = "account-reported", authorName = "Ana") {
    return new KeptMessage({
        messageId: `${accountId}-${n}`, accountId, authorName, text: `${authorName} ${n}`,
        sentAt: Timestamp.fromDate(new Date(AT.getTime() + n * 60_000)), countryId: "fr",
    })
}

function open(admin: FakeAdmin) {
    render(<ReportDetail report={fromTheCard} onBack={() => undefined} clients={admin.clients()}/>)
}

function panel() {
    return within(screen.getByRole("region", {name: "Their messages"}))
}

async function theirs(): Promise<string[]> {
    const list = await panel().findByRole("list", {name: "Their messages, newest first"})
    return within(list).getAllByText(/^Ana \d+$/, {selector: ".line-text"}).map(line => line.textContent ?? "")
}

describe("TheirMessages", () => {
    it("lists what the reported account said, newest first, read by its account id", async () => {
        const admin = new FakeAdmin()
        admin.chat = [said(1), said(1, "account-cleo", "Cleo"), said(2), said(3)]
        open(admin)

        expect(await theirs()).toEqual(["Ana 3", "Ana 2", "Ana 1"])
        expect(admin.askedFor("listMessages")).toEqual([{accountId: "account-reported", limit: 20}])
        expect(screen.queryByRole("region", {name: "The message"})).toBeNull()
    })

    it("reads their earlier messages on asking, from the oldest shown", async () => {
        const admin = new FakeAdmin()
        admin.chat = Array.from({length: 25}, (_, n) => said(n + 1))
        open(admin)

        expect(await theirs()).toHaveLength(20)
        fireEvent.click(panel().getByRole("button", {name: "Show earlier"}))

        await waitFor(async () => expect(await theirs()).toHaveLength(25))
        expect((await theirs()).at(-1)).toBe("Ana 1")
        expect(admin.askedFor("listMessages").at(-1))
            .toEqual({accountId: "account-reported", beforeMessageId: "account-reported-6", limit: 20})
        expect(panel().queryByRole("button", {name: "Show earlier"})).toBeNull()
    })

    it("shows what everyone said around one of their messages, on asking", async () => {
        const admin = new FakeAdmin()
        admin.chat = [said(1, "account-reporter", "Bob"), said(2), said(3, "account-cleo", "Cleo")]
        open(admin)
        await theirs()

        fireEvent.click(panel().getByRole("button", {name: "Show what was said around it"}))

        await panel().findByRole("list", {name: "The conversation"})
        await waitFor(() => expect(panel().queryByText("Loading what was said around it…")).toBeNull())
        const conversation = panel().getByRole("list", {name: "The conversation"})
        expect(within(conversation).getByText("Cleo 3")).toBeTruthy()
        expect(within(conversation).getByText("Bob 1")).toBeTruthy()
        expect(within(conversation).getByText("This message")).toBeTruthy()
        expect(within(conversation).getByText("reporter")).toBeTruthy()
        expect(admin.askedFor("listMessages").slice(1)).toEqual([
            {beforeMessageId: "account-reported-2", limit: 10},
            {afterMessageId: "account-reported-2", limit: 10},
        ])
    })

    it("strikes a message of theirs that was removed", async () => {
        const admin = new FakeAdmin()
        const removed = said(1)
        removed.removedAt = Timestamp.fromDate(AT)
        admin.chat = [removed]
        open(admin)

        await theirs()
        expect(panel().getByText("Ana 1").className).toContain("line-removed")
        expect(panel().getByText(/^Removed at /)).toBeTruthy()
    })

    it("says when nothing they said is kept", async () => {
        const admin = new FakeAdmin()
        admin.chat = [said(1, "account-cleo", "Cleo")]
        open(admin)

        expect(await panel().findByText("Nothing they said in the chat is kept.")).toBeTruthy()
    })

    it("says when their messages cannot be read, and reads them again", async () => {
        const admin = new FakeAdmin()
        admin.chat = [said(1)]
        admin.refuse("listMessages", Code.Unavailable)
        open(admin)

        expect(await panel().findByText("The call failed.")).toBeTruthy()
        admin.refusals.delete("listMessages")
        fireEvent.click(panel().getByRole("button", {name: "Try again"}))

        expect(await theirs()).toEqual(["Ana 1"])
    })
})
