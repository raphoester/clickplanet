import {useCallback, useState} from "react"
import {supportAdmin} from "../api.ts"
import {AccountId} from "../components/AccountId.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {Note, NoteKind} from "../gen/grpc/support/v1/admin_pb.ts"
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

export function NoteList() {
    const [kind, setKind] = useState(NoteKind.UNSPECIFIED)
    const list = useCallback(
        (signal: AbortSignal) => supportAdmin.listNotes({kind}, {signal}).then(answer => answer.notes),
        [kind],
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
                    <p className="muted">{countOf(notes.value.length, "note")}</p>
                    <ol className="cards">
                        {notes.value.map(note => <NoteCard key={note.id} note={note}/>)}
                    </ol>
                </>
            )}
        </section>
    )
}

function NoteCard({note}: {note: Note}) {
    const device = deviceLines(note.context, note.userAgent)

    return (
        <li className="card">
            <div className="card-head">
                <span className={`badge ${BADGES[note.kind] ?? ""}`}>{kindLabel(note.kind)}</span>
                <When at={note.submittedAt}/>
            </div>
            <p className="text">{note.text}</p>
            <Attachments files={note.attachments}/>
            <AccountId id={note.accountId}/>
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
        </li>
    )
}
