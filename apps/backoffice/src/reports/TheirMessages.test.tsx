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
import {AROUND} from "./theirConversations.ts"

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

const AUTHORS: Record<string, string> = {"account-reported": "Ana", "account-reporter": "Bob", "account-cleo": "Cleo"}

// The chat, one message a minute, each line "Ana", "Bob" or "Cleo" for whoever says it: "Bob 1", "Ana 2"…
function chatOf(...who: string[]): KeptMessage[] {
    return who.map((name, i) => {
        const accountId = Object.keys(AUTHORS).find(account => AUTHORS[account] === name) ?? ""
        return new KeptMessage({
            messageId: `m${i + 1}`, accountId, authorName: name, text: `${name} ${i + 1}`,
            sentAt: Timestamp.fromDate(new Date(AT.getTime() + i * 60_000)), countryId: "fr",
        })
    })
}

// Ana's lines, `gap` messages of Bob's between each two.
function apart(count: number, gap: number): KeptMessage[] {
    return chatOf(...Array.from({length: count}, () => ["Ana", ...Array<string>(gap).fill("Bob")]).flat())
}

function open(admin: FakeAdmin) {
    render(<ReportDetail report={fromTheCard} onBack={() => undefined} clients={admin.clients()}/>)
}

function panel() {
    return within(screen.getByRole("region", {name: "Their messages"}))
}

// The text of each line of each conversation shown, once they are.
async function conversations(): Promise<string[][]> {
    const list = await panel().findByRole("list", {name: "Their conversations, newest first"})
    return within(list).getAllByRole("list", {name: "A conversation"}).map(conversation =>
        within(conversation).getAllByText(/^(Ana|Bob|Cleo) \d+$/, {selector: ".line-text"}).map(line => line.textContent ?? ""))
}

describe("TheirMessages", () => {
    it("shows what they said in its conversation, with no click", async () => {
        const admin = new FakeAdmin()
        admin.chat = chatOf("Bob", "Ana", "Cleo")
        open(admin)

        expect(await conversations()).toEqual([["Bob 1", "Ana 2", "Cleo 3"]])
        expect(admin.askedFor("listMessages")).toEqual([
            {accountId: "account-reported", limit: 20},
            {beforeMessageId: "m2", limit: AROUND},
            {afterMessageId: "m2", limit: AROUND},
        ])
        expect(panel().getByText("Ana 2").closest("li")?.className).toContain("line-theirs")
        expect(panel().getByText("reporter")).toBeTruthy()
        expect(screen.queryByRole("region", {name: "The message"})).toBeNull()
    })

    it("shows an exchange as one conversation, read once", async () => {
        const admin = new FakeAdmin()
        admin.chat = chatOf("Bob", "Ana", "Cleo", "Ana", "Bob")
        open(admin)

        expect(await conversations()).toEqual([["Bob 1", "Ana 2", "Cleo 3", "Ana 4", "Bob 5"]])
        expect(admin.askedFor("listMessages")).toHaveLength(3)
    })

    it("keeps conversations far apart apart, the newest first", async () => {
        const admin = new FakeAdmin()
        admin.chat = apart(2, 2 * AROUND + 1)
        open(admin)

        const shown = await conversations()
        expect(shown).toHaveLength(2)
        expect(shown[0]).toContain("Ana 13")
        expect(shown[1]).toContain("Ana 1")
    })

    it("builds three conversations, then the next on asking", async () => {
        const admin = new FakeAdmin()
        admin.chat = apart(5, 2 * AROUND + 1)
        open(admin)

        expect(await conversations()).toHaveLength(3)
        fireEvent.click(panel().getByRole("button", {name: "Show more conversations"}))

        await waitFor(async () => expect(await conversations()).toHaveLength(5))
        expect(panel().queryByRole("button", {name: "Show more conversations"})).toBeNull()
    })

    it("reads more of their messages once every one read is in a conversation", async () => {
        const admin = new FakeAdmin()
        admin.chat = apart(21, 2 * AROUND + 1)
        open(admin)
        await conversations()

        // 3 at a time from the 20 read: 6, 9 … 18, 20, then the 21st, once the next page of theirs is read.
        for (const expected of [6, 9, 12, 15, 18, 20, 21]) {
            fireEvent.click(panel().getByRole("button", {name: "Show more conversations"}))
            await waitFor(async () => expect(await conversations()).toHaveLength(expected))
        }

        const theirs = admin.askedFor("listMessages").filter(ask => (ask as {accountId?: string}).accountId)
        expect(theirs).toEqual([
            {accountId: "account-reported", limit: 20},
            {accountId: "account-reported", beforeMessageId: "m13", limit: 20},
        ])
        expect(panel().queryByRole("button", {name: "Show more conversations"})).toBeNull()
    })

    it("grows a conversation earlier and later, on asking", async () => {
        const admin = new FakeAdmin()
        admin.chat = chatOf(...Array<string>(20).fill("Bob"), "Ana", ...Array<string>(20).fill("Bob"))
        open(admin)

        expect((await conversations())[0]).toHaveLength(2 * AROUND + 1)
        fireEvent.click(panel().getByRole("button", {name: "Show earlier"}))
        await waitFor(async () => expect((await conversations())[0]).toHaveLength(2 * AROUND + 11))
        fireEvent.click(panel().getByRole("button", {name: "Show later"}))
        await waitFor(async () => expect((await conversations())[0]).toHaveLength(2 * AROUND + 21))

        expect(admin.askedFor("listMessages").slice(3)).toEqual([
            {beforeMessageId: "m16", limit: 10},
            {afterMessageId: "m26", limit: 10},
        ])
    })

    it("strikes a message of theirs that was removed", async () => {
        const admin = new FakeAdmin()
        admin.chat = chatOf("Ana")
        admin.chat[0].removedAt = Timestamp.fromDate(AT)
        open(admin)

        await conversations()
        expect(panel().getByText("Ana 1").className).toContain("line-removed")
        expect(panel().getByText(/^Removed at /)).toBeTruthy()
    })

    it("says when nothing they said is kept", async () => {
        const admin = new FakeAdmin()
        admin.chat = chatOf("Cleo")
        open(admin)

        expect(await panel().findByText("Nothing they said in the chat is kept.")).toBeTruthy()
    })

    it("says when their messages cannot be read, and reads them again", async () => {
        const admin = new FakeAdmin()
        admin.chat = chatOf("Ana")
        admin.refuse("listMessages", Code.Unavailable)
        open(admin)

        expect(await panel().findByText("The call failed.")).toBeTruthy()
        admin.refusals.delete("listMessages")
        fireEvent.click(panel().getByRole("button", {name: "Try again"}))

        expect(await conversations()).toEqual([["Ana 1"]])
    })
})
