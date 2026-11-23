// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {Player as Identity} from "../gen/grpc/player/v1/player_pb.ts"
import {Note, NoteAnswer, NoteKind, NoteReply} from "../gen/grpc/support/v1/admin_pb.ts"
import {OpenPlayer} from "../navigation.ts"
import {MAX_NOTES, NotesView} from "./NotesView.tsx"

afterEach(cleanup)

const minutes = (n: number) => Timestamp.fromDate(new Date(AT.getTime() + n * 60_000))

// BraveFox42's bug, new; Ben's idea, answered, then replied to by his mother and by him; Cleo's idea, answered; and
// an old bug whose sender deleted its account, with no address.
function withNotes(): FakeAdmin {
    const admin = new FakeAdmin()
    admin.identities = [
        new Identity({accountId: ACCOUNT, name: "BraveFox42"}),
        new Identity({accountId: "account-ben", name: "Ben"}),
        new Identity({accountId: "account-cleo", name: "Cleo"}),
    ]
    admin.notes = [
        new Note({
            id: "note-ada", kind: NoteKind.BUG, accountId: ACCOUNT, text: "the globe goes black",
            submittedAt: minutes(-60), replyAddress: "ada@example.com",
        }),
        new Note({
            id: "note-ben", kind: NoteKind.SUGGESTION, accountId: "account-ben", text: "more flags",
            submittedAt: minutes(-300), replyAddress: "ben@example.com",
            answers: [new NoteAnswer({text: "Which flags?", sentAt: minutes(-240)})],
            replies: [
                new NoteReply({text: "he plays too much", from: "mum@example.org", receivedAt: minutes(-20)}),
                new NoteReply({text: "Tibet", from: "ben@example.com", receivedAt: minutes(-10)}),
            ],
        }),
        new Note({
            id: "note-cleo", kind: NoteKind.SUGGESTION, accountId: "account-cleo", text: "dark mode",
            submittedAt: minutes(-500), replyAddress: "cleo@example.com",
            answers: [new NoteAnswer({text: "It follows your phone now.", sentAt: minutes(-400)})],
        }),
        new Note({id: "note-old", kind: NoteKind.BUG, text: "it lags", submittedAt: minutes(-900)}),
    ]
    return admin
}

function Inbox({admin, first = ""}: {admin: FakeAdmin, first?: string}) {
    const [opened, setOpened] = useState(first)
    return (
        <>
            <p data-testid="opened">{opened}</p>
            <NotesView opened={opened} onOpen={setOpened} clients={admin.clients()}/>
        </>
    )
}

function show(admin: FakeAdmin, first = "") {
    const openPlayer = vi.fn()
    render(
        <OpenPlayer.Provider value={openPlayer}>
            <Inbox admin={admin} first={first}/>
        </OpenPlayer.Provider>,
    )
    return openPlayer
}

function rows() {
    return within(screen.getByRole("list", {name: "The notes"})).getAllByRole("listitem")
}

function answer(text: string) {
    fireEvent.change(screen.getByRole("textbox"), {target: {value: text}})
}

describe("the notes inbox", () => {
    it("lists the notes waiting for an answer, the last written to first, each with what the player wrote last", async () => {
        const admin = withNotes()
        show(admin)

        expect(await screen.findByRole("button", {name: "Open (2)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "All (4)"})).toBeTruthy()
        const [ben, ada] = rows()
        expect(rows()).toHaveLength(2)
        expect(within(ben).getByText("Ben")).toBeTruthy()
        expect(within(ben).getByText("Tibet")).toBeTruthy()
        expect(within(ben).getByText("Replied")).toBeTruthy()
        expect(within(ada).getByText("BraveFox42")).toBeTruthy()
        expect(within(ada).getByText("the globe goes black")).toBeTruthy()
        expect(within(ada).getByText("New")).toBeTruthy()
        expect(admin.askedFor("listNotes")).toEqual([{openOnly: true, limit: MAX_NOTES}, {limit: MAX_NOTES}])
        expect(screen.getByText("Pick a note on the left.")).toBeTruthy()
    })

    it("lists every note under All, the open ones first, each with where it stands", async () => {
        show(withNotes())

        fireEvent.click(await screen.findByRole("button", {name: "All (4)"}))

        expect(rows().map(row => within(row).getByText(/^(New|Replied|Answered|Closed|No address)$/).textContent))
            .toEqual(["Replied", "New", "Answered", "No address"])
        expect(within(rows()[3]).getByText("no account")).toBeTruthy()
    })

    it("shows the conversation in time order, a reply from another address by that address", async () => {
        show(withNotes(), "note-ben")

        const said = within(await screen.findByRole("list", {name: "What was said"})).getAllByRole("listitem")
        expect(said.map(line => line.querySelector(".name")?.textContent)).toEqual(["Ben", "ClickPlanet", "mum@example.org", "Ben"])
        expect(said.map(line => line.querySelector(".line-text")?.textContent))
            .toEqual(["more flags", "Which flags?", "he plays too much", "Tibet"])
    })

    it("asks before it sends, naming the address, then opens the next open note and says it was sent", async () => {
        const admin = withNotes()
        show(admin, "note-ben")

        const panel = within(await screen.findByRole("region", {name: "Answer"}))
        expect((panel.getByRole("button", {name: "Send"}) as HTMLButtonElement).disabled).toBe(true)
        answer("  Tibet is on the list.\n")
        fireEvent.click(panel.getByRole("button", {name: "Send"}))
        expect(panel.getByText("Send this answer to ben@example.com?")).toBeTruthy()
        fireEvent.click(panel.getByRole("button", {name: "Yes, send"}))

        expect((await screen.findByRole("status")).textContent).toBe("Ben: Sent to ben@example.com.")
        await waitFor(() => expect(screen.getByTestId("opened").textContent).toBe("note-ada"))
        expect(admin.askedFor("answerNote")).toEqual([{noteId: "note-ben", text: "Tibet is on the list."}])
        expect(screen.getByRole("button", {name: "Open (1)"})).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "All (4)"}))
        expect(within(rows()[1]).getByText("Answered")).toBeTruthy()
    })

    it("keeps the text typed when the mail did not go, and says to send it again", async () => {
        const admin = withNotes()
        admin.refuse("answerNote", Code.Unavailable)
        show(admin, "note-ben")

        const panel = within(await screen.findByRole("region", {name: "Answer"}))
        answer("Tibet is on the list.")
        fireEvent.click(panel.getByRole("button", {name: "Send"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, send"}))

        expect((await panel.findByRole("alert")).textContent).toBe("The mail did not go. Send it again.")
        expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("Tibet is on the list.")
        expect(screen.getByTestId("opened").textContent).toBe("note-ben")
        expect(screen.getByRole("button", {name: "Open (2)"})).toBeTruthy()
    })

    it("closes a note with no answer, asking first, then opens the first one left", async () => {
        const admin = withNotes()
        show(admin, "note-ada")

        const panel = within(await screen.findByRole("region", {name: "Answer"}))
        fireEvent.click(panel.getByRole("button", {name: "Close without answering"}))
        expect(panel.getByText("Close this note without answering?")).toBeTruthy()
        fireEvent.click(panel.getByRole("button", {name: "Yes, close without answering"}))

        expect((await screen.findByRole("status")).textContent).toBe("BraveFox42: Closed with no answer.")
        await waitFor(() => expect(screen.getByTestId("opened").textContent).toBe("note-ben"))
        expect(admin.askedFor("closeNote")).toEqual([{noteId: "note-ada"}])
        expect(admin.askedFor("answerNote")).toEqual([])
        fireEvent.click(screen.getByRole("button", {name: "All (4)"}))
        expect(within(rows()[1]).getByText("Closed")).toBeTruthy()
    })

    it("says when no note waits once the last one is done", async () => {
        const admin = withNotes()
        admin.notes = admin.notes.filter(note => note.id === "note-ada")
        show(admin, "note-ada")

        const panel = within(await screen.findByRole("region", {name: "Answer"}))
        answer("Fixed, thanks.")
        fireEvent.click(panel.getByRole("button", {name: "Send"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, send"}))

        expect(await screen.findByText("No open note. All done.")).toBeTruthy()
        expect(screen.getByTestId("opened").textContent).toBe("")
        expect(screen.getByRole("status").textContent).toBe("BraveFox42: Sent to ada@example.com.")
    })

    it("answers a note answered already again, with nothing to close", async () => {
        show(withNotes(), "note-cleo")

        const panel = within(await screen.findByRole("region", {name: "Answer"}))
        expect(panel.getByRole("button", {name: "Send"})).toBeTruthy()
        expect(panel.queryByRole("button", {name: "Close without answering"})).toBeNull()
    })

    it("says a note with no address cannot be answered, and has no answer box", async () => {
        show(withNotes(), "note-old")

        expect(await screen.findByText("No email address is kept for this note: it cannot be answered.")).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Answer"})).toBeNull()
        expect(screen.queryByRole("textbox")).toBeNull()
    })

    it("names who sent the note, with its address, and opens the sender on the player page", async () => {
        const openPlayer = show(withNotes(), "note-ada")

        const note = within(await screen.findByRole("region", {name: "Note"}))
        expect(note.getByText(ACCOUNT)).toBeTruthy()
        expect(note.getByText("ada@example.com")).toBeTruthy()
        fireEvent.click(note.getByRole("button", {name: "Open player"}))

        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: ACCOUNT}, name: "BraveFox42"})
    })

    it("has no player to open for a note whose sender deleted its account", async () => {
        show(withNotes(), "note-old")

        const note = within(await screen.findByRole("region", {name: "Note"}))
        expect(note.getAllByText("no account").length).toBeGreaterThan(0)
        expect(note.queryByRole("button", {name: "Open player"})).toBeNull()
    })

    it("still lists the notes when the names cannot be read, and says why", async () => {
        const admin = withNotes()
        admin.refuse("getPlayers", Code.Unavailable)
        show(admin)

        expect(await screen.findByText("The names could not be read. The call failed: refused")).toBeTruthy()
        expect(within(rows()[0]).getByText("no name")).toBeTruthy()
    })
})
