import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {Letter, LetterMail, NoteReply} from "../gen/grpc/support/v1/admin_pb.ts"
import {letterRefusalOf, threadOf, waitsOf, writeLetter} from "./letters.ts"

const at = (minutes: number) => Timestamp.fromDate(new Date(AT.getTime() + minutes * 60_000))
const letter = (part: Partial<Letter> = {}) => new Letter({
    id: "letter-1", submissionId: "sub-catalonia", accountId: ACCOUNT, address: "ada@example.com", subject: "Your flag Catalonia",
    mails: [new LetterMail({text: "Which Catalonia?", sentAt: at(0)})], ...part,
})

describe("a letter", () => {
    it("reads as its mails, the player's replies and its close, by time", () => {
        const read = letter({
            mails: [new LetterMail({text: "Which Catalonia?", sentAt: at(0)}), new LetterMail({text: "Thanks!", sentAt: at(20)})],
            replies: [new NoteReply({text: "the spanish one", from: "ada@example.com", receivedAt: at(10)})],
            closedAt: at(30),
        })
        expect(threadOf(read).map(entry => entry.kind)).toEqual(["mail", "reply", "mail", "closed"])
    })

    it("waits for the operator once the player replied, until a mail or a close after it", () => {
        const replied = letter({replies: [new NoteReply({text: "the spanish one", receivedAt: at(10)})]})
        expect(waitsOf(letter())).toBe(false)
        expect(waitsOf(replied)).toBe(true)
        expect(waitsOf(new Letter({...replied, closedAt: at(11)}))).toBe(false)
        expect(waitsOf(new Letter({...replied, mails: [...replied.mails, new LetterMail({text: "Thanks!", sentAt: at(12)})]}))).toBe(false)
    })
})

describe("writing a letter", () => {
    it("sends the letter about a submission to the account's verified address, then one more mail in it", async () => {
        const admin = new FakeAdmin()
        const writing = {submissionId: "sub-catalonia", accountId: ACCOUNT, subject: " Your flag Catalonia "}
        const first = await writeLetter(admin, writing, " Which Catalonia? ")
        expect(first.address).toBe("ada@example.com")
        expect(first.subject).toBe("Your flag Catalonia")
        expect(first.submissionId).toBe("sub-catalonia")
        const again = await writeLetter(admin, {letterId: first.id}, "Thanks!")
        expect(again.mails.map(mail => mail.text)).toEqual(["Which Catalonia?", "Thanks!"])
        expect(admin.askedFor("writeLetter")).toEqual([
            {submissionId: "sub-catalonia", accountId: ACCOUNT, subject: "Your flag Catalonia", text: "Which Catalonia?"},
            {letterId: first.id, text: "Thanks!"},
        ])
    })

    it("says the server's refusals in words", async () => {
        const admin = new FakeAdmin()
        const refused = await writeLetter(admin, {submissionId: "sub-beach", accountId: "account-gone", subject: "Hi"}, "Hello")
            .catch(err => err)
        expect((refused as ConnectError).code).toBe(Code.FailedPrecondition)
        expect(letterRefusalOf("write", refused))
            .toBe("No mail can go: the player has no verified address, or the letter is over 90 days old.")
        expect(letterRefusalOf("write", new ConnectError("smtp", Code.Unavailable))).toBe("The mail did not go, and nothing was kept. Send it again.")
        const long = await writeLetter(admin, {submissionId: "sub-catalonia", accountId: ACCOUNT, subject: "x".repeat(201)}, "Hello")
            .catch(err => err)
        expect(letterRefusalOf("write", long)).toBe("The server refused it: a subject of 1 to 200 characters")
        await writeLetter(admin, {submissionId: "sub-catalonia", accountId: ACCOUNT, subject: "Your flag Catalonia"}, "Hello")
        const twice = await writeLetter(admin, {submissionId: "sub-catalonia", accountId: ACCOUNT, subject: "Your flag Catalonia"}, "Hi")
            .catch(err => err)
        expect((twice as ConnectError).code).toBe(Code.AlreadyExists)
        expect(letterRefusalOf("write", twice)).toBe("This submission has its letter already. Reload the page to write in it.")
    })
})
