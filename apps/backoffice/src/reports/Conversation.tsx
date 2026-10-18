import {useState} from "react"
import {clockOf, timeOf} from "../components/time.ts"
import {GetMessageResponse, KeptMessage, KeptQuote} from "../gen/grpc/chat/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"

// How many of the messages on each side of the one reported show before the operator asks for the rest.
const SHOWN_AROUND = 10

// The accounts a report names, so the conversation marks what each of them said.
export type Parties = {reporter: string, reported: string}

type Props = {
    message: GetMessageResponse
    removedAt: Date | undefined
    parties: Parties
}

// The message reported among those sent just before and just after it, oldest first, as the chat showed them.
export function Conversation({message, removedAt, parties}: Props) {
    const [allBefore, setAllBefore] = useState(false)
    const [allAfter, setAllAfter] = useState(false)
    const before = allBefore ? message.before : message.before.slice(-SHOWN_AROUND)
    const after = allAfter ? message.after : message.after.slice(0, SHOWN_AROUND)
    const day = message.sentAt?.toDate()

    return (
        <div className="conversation">
            {message.before.length === 0 && <p className="hint">Nothing said before it is kept.</p>}
            {before.length < message.before.length && (
                <button type="button" className="button-small" onClick={() => setAllBefore(true)}>
                    Show {message.before.length - before.length} earlier
                </button>
            )}
            <ol className="lines" aria-label="The conversation">
                {before.map(line => <Line key={line.messageId} line={line} parties={parties} day={day}/>)}
                <Line line={keptOf(message)} parties={parties} day={day} removedAt={removedAt} reported/>
                {after.map(line => <Line key={line.messageId} line={line} parties={parties} day={day}/>)}
            </ol>
            {after.length < message.after.length && (
                <button type="button" className="button-small" onClick={() => setAllAfter(true)}>
                    Show {message.after.length - after.length} later
                </button>
            )}
            {message.after.length === 0 && <p className="hint">Nothing was said after it.</p>}
        </div>
    )
}

function keptOf(message: GetMessageResponse): KeptMessage {
    return new KeptMessage({
        messageId: message.messageId,
        accountId: message.accountId,
        authorName: message.authorName,
        text: message.text,
        sentAt: message.sentAt,
        countryId: message.countryId,
        removedAt: message.removedAt,
        replyTo: message.replyTo,
    })
}

type LineProps = {
    line: KeptMessage
    parties: Parties
    // The day of the message reported: a line sent on another one shows its date too, as the reported one does.
    day: Date | undefined
    reported?: boolean
    removedAt?: Date
}

function Line({line, parties, day, reported = false, removedAt}: LineProps) {
    const removed = removedAt ?? line.removedAt?.toDate()
    const party = partyOf(line.accountId, parties)

    return (
        <li className={reported ? "line line-reported" : "line"} aria-current={reported ? "true" : undefined}>
            <div className="line-head">
                {reported && <span className="badge badge-report">Reported message</span>}
                <Author name={line.authorName} accountId={line.accountId}/>
                {party !== undefined && <span className={`tag tag-${party}`}>{party}</span>}
                <span className="muted">
                    {line.countryId !== "" && <>{line.countryId.toUpperCase()} · </>}
                    <Sent at={line.sentAt?.toDate()} day={reported ? undefined : day}/>
                </span>
            </div>
            {line.replyTo !== undefined && <Quote quote={line.replyTo}/>}
            <p className={removed === undefined ? "line-text" : "line-text line-removed"}>{line.text}</p>
            {removed !== undefined && !reported && <p className="hint">Removed at {timeOf(removed)}.</p>}
        </li>
    )
}

function partyOf(accountId: string, parties: Parties): "reporter" | "reported" | undefined {
    if (accountId === "") return undefined
    if (accountId === parties.reported) return "reported"
    if (accountId === parties.reporter) return "reporter"
    return undefined
}

// The author opens on the player page, by its account id, where the page can open one.
function Author({name, accountId}: {name: string, accountId: string}) {
    const openPlayer = useOpenPlayer()
    const shown = name === "" ? "no name" : name
    if (openPlayer === undefined || accountId === "") return <span className="name">{shown}</span>
    return (
        <button
            type="button"
            className="line-author"
            title="Open on the player page"
            onClick={() => openPlayer({caller: {accountId}, name})}
        >
            {shown}
        </button>
    )
}

function Sent({at, day}: {at: Date | undefined, day: Date | undefined}) {
    if (at === undefined) return null
    const sameDay = day !== undefined && at.toDateString() === day.toDateString()
    return <time dateTime={at.toISOString()} title={timeOf(at)}>{sameDay ? clockOf(at) : timeOf(at)}</time>
}

function Quote({quote}: {quote: KeptQuote}) {
    const removed = quote.removedAt !== undefined
    return (
        <p className="line-quote">
            In reply to <span className="name">{quote.authorName === "" ? "no name" : quote.authorName}</span>:{" "}
            <span className={removed ? "line-removed" : undefined}>{quote.text}</span>
            {removed && " (removed)"}
        </p>
    )
}
