import {useCallback, useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Person} from "../components/Person.tsx"
import {When} from "../components/When.tsx"
import {Note, NoteKind} from "../gen/grpc/support/v1/admin_pb.ts"
import {Author} from "../gen/grpc/player/v1/internal_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {namesOf} from "../players/actions.ts"
import {useCall} from "../useCall.ts"
import {Attachments} from "./Attachments.tsx"
import {countOf, deviceLines, kindLabel} from "./labels.ts"

const FILTERS: {kind: NoteKind, label: string}[] = [
    {kind: NoteKind.UNSPECIFIED, label: "All"},
    {kind: NoteKind.SUGGESTION, label: "Ideas"},
    {kind: NoteKind.BUG, label: "Bugs"},
]

const BADGES: Partial<Record<NoteKind, string>> = {
    [NoteKind.SUGGESTION]: "badge-idea",
    [NoteKind.BUG]: "badge-bug",
}

// Each note with who sent it: notes carry an account id alone, so the page asks the player module for the names.
export function NoteList({clients = CLIENTS}: {clients?: Clients}) {
    const [kind, setKind] = useState(NoteKind.UNSPECIFIED)
    const list = useCallback(
        async (signal: AbortSignal) => {
            const {notes} = await clients.support.listNotes({kind}, {signal})
            return {notes, ...await namesOf(clients.player, notes.map(note => note.accountId), signal)}
        },
        [clients, kind],
    )
    const [notes, again] = useCall(list)

    return (
        <section aria-label="Notes">
            <div className="toolbar">
                <div className="segmented" role="group" aria-label="Kind">
                    {FILTERS.map(filter => (
                        <button
                            key={filter.kind}
                            type="button"
                            aria-pressed={kind === filter.kind}
                            onClick={() => setKind(filter.kind)}
                        >
                            {filter.label}
                        </button>
                    ))}
                </div>
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>

            {notes.state === "loading" && <p className="muted">Loading…</p>}
            {notes.state === "failed" && <FailureNotice failure={notes} onRetry={again}/>}
            {notes.state === "done" && (
                <>
                    <p className="muted">{countOf(notes.value.notes.length, "note")}</p>
                    {notes.value.unnamed !== undefined && (
                        <p className="action-refused">The names could not be read. {notes.value.unnamed}</p>
                    )}
                    <ol className="cards">
                        {notes.value.notes.map(note => (
                            <NoteCard key={note.id} note={note} sender={notes.value.authors.get(note.accountId)}/>
                        ))}
                    </ol>
                </>
            )}
        </section>
    )
}

function NoteCard({note, sender}: {note: Note, sender: Author | undefined}) {
    const openPlayer = useOpenPlayer()
    const device = deviceLines(note.context, note.userAgent)
    const name = sender?.name ?? ""

    return (
        <li className="card">
            <div className="card-head">
                <span className={`badge ${BADGES[note.kind] ?? ""}`}>{kindLabel(note.kind)}</span>
                <When at={note.submittedAt}/>
            </div>
            <p className="text">{note.text}</p>
            <Attachments files={note.attachments}/>
            <dl className="people">
                <Person label="From" name={name} id={note.accountId}/>
            </dl>
            {device.length > 0 && (
                <details className="device">
                    <summary>Device</summary>
                    <dl>
                        {device.map(line => (
                            <div key={line.label}>
                                <dt>{line.label}</dt>
                                <dd>{line.value}</dd>
                            </div>
                        ))}
                    </dl>
                </details>
            )}
            {openPlayer !== undefined && note.accountId !== "" && (
                <div className="card-foot">
                    <span/>
                    <button type="button" className="button" onClick={() => openPlayer({caller: {accountId: note.accountId}, name})}>
                        Open player
                    </button>
                </div>
            )}
        </li>
    )
}
