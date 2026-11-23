import {Ago} from "../components/When.tsx"
import {Note} from "../gen/grpc/support/v1/admin_pb.ts"
import {kindBadge, kindLabel, senderLabel} from "./labels.ts"
import {lastAtOf, lastWordsOf, statusOf} from "./notes.ts"

type Props = {
    notes: Note[]
    isOpen: (note: Note) => boolean
    nameOf: (note: Note) => string
    opened: string
    onOpen: (noteId: string) => void
}

// One row a note: its kind, who sent it, how long ago it was last written to, and what the player wrote last.
export function NoteList({notes, isOpen, nameOf, opened, onOpen}: Props) {
    return (
        <ol className="rows" aria-label="The notes">
            {notes.map(note => (
                <li key={note.id}>
                    <button
                        type="button"
                        className="row"
                        aria-current={note.id === opened ? "true" : undefined}
                        onClick={() => onOpen(note.id)}
                    >
                        <span className="row-head">
                            <span className={kindBadge(note.kind)}>{kindLabel(note.kind)}</span>
                            <span className="name">{senderLabel(nameOf(note), note.accountId)}</span>
                            <span className="row-end"><Ago at={lastAtOf(note)}/></span>
                        </span>
                        <span className="row-text">{lastWordsOf(note)}</span>
                        <span className="row-sub muted">{statusOf(note, isOpen(note))}</span>
                    </button>
                </li>
            ))}
        </ol>
    )
}
