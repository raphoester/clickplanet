import {SupportAdmin} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {Person} from "../components/Person.tsx"
import {When} from "../components/When.tsx"
import {Note} from "../gen/grpc/support/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {Acted} from "./actions.ts"
import {AnswerPanel} from "./AnswerPanel.tsx"
import {Attachments} from "./Attachments.tsx"
import {deviceLines, kindBadge, kindLabel, senderLabel} from "./labels.ts"
import {conversationOf, Entry} from "./notes.ts"

type Props = {
    note: Note
    name: string
    open: boolean
    support: SupportAdmin
    onBack: () => void
    onActed: (acted: Acted) => void
}

// One note, top down: who sent it and from what, what was said since, then the answer.
export function NoteDetail({note, name, open, support, onBack, onActed}: Props) {
    const openPlayer = useOpenPlayer()
    const device = deviceLines(note.context, note.userAgent)

    return (
        <section className="detail" aria-label="Note">
            <button type="button" className="button-small back" onClick={onBack}>← All notes</button>
            <article className="card note-head">
                <div className="card-head">
                    <span className={kindBadge(note.kind)}>{kindLabel(note.kind)}</span>
                    <When at={note.submittedAt}/>
                </div>
                <dl className="people">
                    <Person label="From" name={name} id={note.accountId}/>
                    {note.replyAddress !== "" && (
                        <div>
                            <dt>Email</dt>
                            <dd><Copyable text={note.replyAddress}/></dd>
                        </div>
                    )}
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
            </article>
            <section className="panel" aria-label="Conversation">
                <h2>Conversation</h2>
                <ol className="lines" aria-label="What was said">
                    {conversationOf(note).map((entry, n) => (
                        <Said key={`${entry.kind}-${n}`} entry={entry} note={note} name={senderLabel(name, note.accountId)}/>
                    ))}
                </ol>
            </section>
            {note.replyAddress === ""
                ? <p className="muted">No email address is kept for this note: it cannot be answered.</p>
                : <AnswerPanel note={note} open={open} support={support} onActed={onActed}/>}
        </section>
    )
}

function Said({entry, note, name}: {entry: Entry, note: Note, name: string}) {
    switch (entry.kind) {
        case "note":
            return (
                <li className="line">
                    <LineHead who={name} at={entry.at}/>
                    <p className="line-text">{entry.text}</p>
                    <Attachments files={note.attachments}/>
                </li>
            )
        case "answer":
            return (
                <li className="line line-answer">
                    <LineHead who="ClickPlanet" at={entry.at}/>
                    <p className="line-text">{entry.text}</p>
                </li>
            )
        case "reply":
            return (
                <li className="line">
                    <LineHead who={entry.from !== "" && entry.from !== note.replyAddress ? entry.from : name} at={entry.at}/>
                    <p className="line-text">{entry.text}</p>
                </li>
            )
        case "closed":
            return (
                <li className="line line-closed">
                    <LineHead who="Closed with no answer" at={entry.at}/>
                </li>
            )
    }
}

// Who said a line of a conversation, and when.
export function LineHead({who, at}: {who: string, at: Entry["at"]}) {
    return (
        <div className="line-head">
            <span className="name">{who}</span>
            <When at={at}/>
        </div>
    )
}
