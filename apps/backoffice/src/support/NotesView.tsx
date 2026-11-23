import {useCallback, useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Note} from "../gen/grpc/support/v1/admin_pb.ts"
import {namesOf} from "../players/actions.ts"
import {useCall} from "../useCall.ts"
import {Acted} from "./actions.ts"
import {senderLabel} from "./labels.ts"
import {NoteDetail} from "./NoteDetail.tsx"
import {NoteList} from "./NoteList.tsx"
import {latestFirst} from "./notes.ts"

// The server's most.
export const MAX_NOTES = 500

type Props = {
    // The note open, by its id, or "" for none.
    opened: string
    onOpen: (noteId: string) => void
    clients?: Clients
}

// An inbox: the notes on the left, the one open on the right. An answer or a close opens the next open one.
export function NotesView({opened, onOpen, clients = CLIENTS}: Props) {
    const {support, player} = clients
    const [openOnly, setOpenOnly] = useState(true)
    const list = useCallback(async (signal: AbortSignal) => {
        const [open, all] = await Promise.all([
            support.listNotes({openOnly: true, limit: MAX_NOTES}, {signal}),
            support.listNotes({limit: MAX_NOTES}, {signal}),
        ])
        const read = new Set(all.notes.map(note => note.id))
        const notes = [...all.notes, ...open.notes.filter(note => !read.has(note.id))]
        const named = await namesOf(player, notes.map(note => note.accountId), signal)
        return {notes, open: new Set(open.notes.map(note => note.id)), ...named}
    }, [support, player])
    const [call, again] = useCall(list)
    // The notes answered or closed on this page since the list was read: none waits any more.
    const [acted, setActed] = useState(new Map<string, Note>())
    const [last, setLast] = useState<{who: string, said: string} | undefined>(undefined)

    if (call.state === "loading") return <p className="muted">Loading…</p>
    if (call.state === "failed") return <FailureNotice failure={call} onRetry={again}/>

    const {identities, unnamed} = call.value
    const notes = call.value.notes.map(note => acted.get(note.id) ?? note)
    const isOpen = (note: Note) => call.value.open.has(note.id) && !acted.has(note.id)
    const open = latestFirst(notes.filter(isOpen))
    const shown = openOnly ? open : [...open, ...latestFirst(notes.filter(note => !isOpen(note)))]
    const note = notes.find(each => each.id === opened)
    const nameOf = (each: Note) => identities.get(each.accountId)?.name ?? ""

    const done = (did: Acted) => {
        setActed(was => new Map([...was, [did.note.id, did.note]]))
        setLast({who: senderLabel(nameOf(did.note), did.note.accountId), said: did.said})
        // The next open note below the one done, or else the first one left.
        const left = open.filter(each => each.id !== did.note.id)
        const at = open.findIndex(each => each.id === opened)
        onOpen((left.find(each => open.indexOf(each) > at) ?? left[0])?.id ?? "")
    }
    // A note picked by hand: what the last answer or close did is no longer the news.
    const pick = (noteId: string) => {
        setLast(undefined)
        onOpen(noteId)
    }
    const reload = () => {
        setActed(new Map())
        setLast(undefined)
        again()
    }

    return (
        <div className={opened === "" ? "inbox" : "inbox inbox-open"}>
            <section className="inbox-list" aria-label="Notes">
                <div className="toolbar">
                    <div className="segmented" role="group" aria-label="Which notes">
                        <button type="button" aria-pressed={openOnly} onClick={() => setOpenOnly(true)}>Open ({open.length})</button>
                        <button type="button" aria-pressed={!openOnly} onClick={() => setOpenOnly(false)}>All ({notes.length})</button>
                    </div>
                    <button type="button" className="button-small" onClick={reload}>Reload</button>
                </div>
                {unnamed !== undefined && <p className="action-refused">The names could not be read. {unnamed}</p>}
                {shown.length === 0
                    ? <p className="muted">{openOnly ? "No open note. All done." : "No note kept."}</p>
                    : <NoteList notes={shown} isOpen={isOpen} nameOf={nameOf} opened={opened} onOpen={pick}/>}
            </section>
            <div className="inbox-detail">
                {last !== undefined && (
                    <p className="done-banner" role="status">
                        <strong>{last.who}</strong>: {last.said}
                    </p>
                )}
                {note !== undefined && (
                    <NoteDetail
                        key={note.id}
                        note={note}
                        name={nameOf(note)}
                        open={isOpen(note)}
                        support={support}
                        onBack={() => pick("")}
                        onActed={done}
                    />
                )}
                {note === undefined && opened !== "" && <p className="muted">This note is not kept any more.</p>}
                {note === undefined && opened === "" && <p className="muted inbox-pick">Pick a note on the left.</p>}
            </div>
        </div>
    )
}
