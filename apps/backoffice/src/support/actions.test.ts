import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {Note, NoteAnswer} from "../gen/grpc/support/v1/admin_pb.ts"
import {answerNote, closeNote, refusalOf} from "./actions.ts"

const note = new Note({id: "note-1", text: "more flags", replyAddress: "ada@example.com"})

describe("answerNote", () => {
    it("sends the answer trimmed, its lines kept, and adds it to the note", async () => {
        const admin = new FakeAdmin()

        const acted = await answerNote(admin, note, "\n Thanks!\nWhich flags? \n")

        expect(admin.askedFor("answerNote")).toEqual([{noteId: "note-1", text: "Thanks!\nWhich flags?"}])
        expect(acted.note.answers).toEqual([new NoteAnswer({text: "Thanks!\nWhich flags?", sentAt: Timestamp.fromDate(AT)})])
        expect(acted.said).toBe("Sent to ada@example.com.")
    })

    it("says what each refusal means", async () => {
        const admin = new FakeAdmin()
        const refusal = async (code: Code) => {
            admin.refuse("answerNote", code)
            return refusalOf("answer", await answerNote(admin, note, "hi").catch((e: unknown) => e))
        }

        expect(await refusal(Code.Unavailable)).toBe("The mail did not go. Send it again.")
        expect(await refusal(Code.FailedPrecondition)).toBe("This note has no address to answer at.")
        expect(await refusal(Code.NotFound)).toBe("This note is no longer kept.")
    })
})

describe("closeNote", () => {
    it("closes the note and says when", async () => {
        const admin = new FakeAdmin()

        const acted = await closeNote(admin, note)

        expect(admin.askedFor("closeNote")).toEqual([{noteId: "note-1"}])
        expect(acted.note.closedAt?.toDate()).toEqual(AT)
        expect(acted.said).toBe("Closed with no answer.")
    })
})
