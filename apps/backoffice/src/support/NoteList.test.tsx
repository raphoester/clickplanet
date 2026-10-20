// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {ACCOUNT, FakeAdmin} from "../fakeAdmin.ts"
import {NoteKind} from "../gen/grpc/support/v1/admin_pb.ts"
import {OpenPlayer} from "../navigation.ts"
import {NoteList} from "./NoteList.tsx"

afterEach(cleanup)

function show(admin: FakeAdmin, openPlayer = vi.fn()) {
    render(
        <OpenPlayer.Provider value={openPlayer}>
            <NoteList clients={admin.clients()}/>
        </OpenPlayer.Provider>,
    )
    return openPlayer
}

describe("the notes", () => {
    it("names who sent each note, beside its account id", async () => {
        const admin = new FakeAdmin()
        show(admin)

        expect(await screen.findByText("2 notes")).toBeTruthy()
        const [bug] = screen.getAllByRole("listitem")
        expect(within(bug).getByText("BraveFox42")).toBeTruthy()
        expect(within(bug).getByText(ACCOUNT)).toBeTruthy()
        expect(admin.askedFor("getAuthors")).toEqual([{accountIds: [ACCOUNT]}])
    })

    it("opens the sender on the player page", async () => {
        const admin = new FakeAdmin()
        const openPlayer = show(admin)
        await screen.findByText("2 notes")

        fireEvent.click(within(screen.getAllByRole("listitem")[0]).getByRole("button", {name: "Open player"}))

        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: ACCOUNT}, name: "BraveFox42"})
    })

    it("has no player to open for a note whose sender deleted its account", async () => {
        const admin = new FakeAdmin()
        show(admin)
        await screen.findByText("2 notes")

        const idea = screen.getAllByRole("listitem")[1]
        expect(within(idea).getByText("no account")).toBeTruthy()
        expect(within(idea).queryByRole("button", {name: "Open player"})).toBeNull()
    })

    it("still lists the notes when the names cannot be read, and says why", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getAuthors", Code.Unavailable)
        const openPlayer = show(admin)

        expect(await screen.findByText("The names could not be read. The call failed: refused")).toBeTruthy()
        const bug = screen.getAllByRole("listitem")[0]
        expect(within(bug).getByText("no name")).toBeTruthy()
        fireEvent.click(within(bug).getByRole("button", {name: "Open player"}))
        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: ACCOUNT}, name: ""})
    })

    it("asks for the kind picked", async () => {
        const admin = new FakeAdmin()
        show(admin)
        await screen.findByText("2 notes")

        fireEvent.click(screen.getByRole("button", {name: "Bugs"}))

        expect(await screen.findByText("1 note")).toBeTruthy()
        expect(admin.askedFor("listNotes").at(-1)).toEqual({kind: NoteKind.BUG})
    })
})
