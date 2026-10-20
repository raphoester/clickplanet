import {useCallback, useState} from "react"
import {ChatAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {clockOf, timeOf} from "../components/time.ts"
import {KeptMessage, KeptQuote} from "../gen/grpc/chat/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {Page, pageOf, Parties} from "./chatPages.ts"

// How many messages the first page on each side of the reported one holds, and each page asked for after.
const FIRST_PAGE = 10
const NEXT_PAGE = 20

export const CHAT_GONE = "That part of the chat is no longer kept."

type Props = {
    chat: ChatAdmin
    anchor: KeptMessage
    removedAt: Date | undefined
    parties: Parties
    // What the anchor is to the operator: "Reported message", "Their message".
    mark: string
}

// A message among the pages of the chat just before and just after it, oldest first, the message itself shown while
// they load.
export function Conversation({chat, anchor, removedAt, parties, mark}: Props) {
    const messageId = anchor.messageId
    const first = useCallback(
        (signal: AbortSignal) => Promise.all([
            pageOf(chat, {beforeMessageId: messageId}, FIRST_PAGE, signal),
            pageOf(chat, {afterMessageId: messageId}, FIRST_PAGE, signal),
        ]).then(([before, after]) => ({before, after})),
        [chat, messageId],
    )
    const [call, again] = useCall(first)
    const reported = {line: anchor, removedAt, mark}

    if (call.state === "done") {
        return <Pages chat={chat} first={call.value} reported={reported} parties={parties}/>
    }
    return (
        <div className="conversation">
            <Lines before={[]} reported={reported} after={[]} parties={parties}/>
            {call.state === "loading" && <p className="hint">Loading what was said around it…</p>}
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
        </div>
    )
}

type Reported = {line: KeptMessage, removedAt: Date | undefined, mark: string}

type PagesProps = {
    chat: ChatAdmin
    first: {before: Page, after: Page}
    reported: Reported
    parties: Parties
}

function Pages({chat, first, reported, parties}: PagesProps) {
    const [before, setBefore] = useState(first.before)
    const [after, setAfter] = useState(first.after)
    const oldest = before.messages[0]?.messageId ?? reported.line.messageId
    const newest = after.messages.at(-1)?.messageId ?? reported.line.messageId

    return (
        <div className="conversation">
            {before.messages.length === 0 && <p className="hint">Nothing said before it is kept.</p>}
            {before.more && (
                <More
                    label="Show earlier"
                    load={() => pageOf(chat, {beforeMessageId: oldest}, NEXT_PAGE)}
                    onLoaded={page => setBefore(shown => ({messages: [...page.messages, ...shown.messages], more: page.more}))}
                />
            )}
            <Lines before={before.messages} reported={reported} after={after.messages} parties={parties}/>
            {after.more && (
                <More
                    label="Show later"
                    load={() => pageOf(chat, {afterMessageId: newest}, NEXT_PAGE)}
                    onLoaded={page => setAfter(shown => ({messages: [...shown.messages, ...page.messages], more: page.more}))}
                />
            )}
            {after.messages.length === 0 && <p className="hint">Nothing was said after it.</p>}
        </div>
    )
}

type MoreStep = {step: "idle"} | {step: "running"} | {step: "refused", words: string}

// The next page, on a click: a read, so it asks nothing first.
export function More({label, load, onLoaded}: {label: string, load: () => Promise<Page>, onLoaded: (page: Page) => void}) {
    const [step, setStep] = useState<MoreStep>({step: "idle"})

    const run = () => {
        setStep({step: "running"})
        load().then(
            page => {
                setStep({step: "idle"})
                onLoaded(page)
            },
            (err: unknown) => setStep({step: "refused", words: wordsOf(err, {notFound: CHAT_GONE})}),
        )
    }

    return (
        <div className="more">
            <button type="button" className="button-small" disabled={step.step === "running"} onClick={run}>
                {step.step === "running" ? "Loading…" : label}
            </button>
            {step.step === "refused" && <p className="action-refused" role="alert">{step.words}</p>}
        </div>
    )
}

type LinesProps = {
    before: KeptMessage[]
    reported: Reported
    after: KeptMessage[]
    parties: Parties
}

function Lines({before, reported, after, parties}: LinesProps) {
    const day = reported.line.sentAt?.toDate()
    return (
        <ol className="lines" aria-label="The conversation">
            {before.map(line => <Line key={line.messageId} line={line} parties={parties} day={day}/>)}
            <Line line={reported.line} parties={parties} day={undefined} removedAt={reported.removedAt} mark={reported.mark}/>
            {after.map(line => <Line key={line.messageId} line={line} parties={parties} day={day}/>)}
        </ol>
    )
}

type LineProps = {
    line: KeptMessage
    // Who the report names, to tag what each said; none when every line is one party's.
    parties: Parties | undefined
    // The anchor's day: a line sent on another one shows its date too, as the anchor does.
    day: Date | undefined
    // The anchor's mark: the line the conversation is around.
    mark?: string
    removedAt?: Date
}

function Line(props: LineProps) {
    const anchor = props.mark !== undefined
    return (
        <li className={anchor ? "line line-reported" : "line"} aria-current={anchor ? "true" : undefined}>
            <LineBody {...props}/>
        </li>
    )
}

export function LineBody({line, parties, day, mark, removedAt}: LineProps) {
    const removed = removedAt ?? line.removedAt?.toDate()
    const party = partyOf(line.accountId, parties)

    return (
        <>
            <div className="line-head">
                {mark !== undefined && <span className="badge badge-report">{mark}</span>}
                <Author name={line.authorName} accountId={line.accountId}/>
                {party !== undefined && <span className={`tag tag-${party}`}>{party}</span>}
                <span className="muted">
                    {line.countryId !== "" && <>{line.countryId.toUpperCase()} · </>}
                    <Sent at={line.sentAt?.toDate()} day={day}/>
                </span>
            </div>
            {line.replyTo !== undefined && <Quote quote={line.replyTo}/>}
            <p className={removed === undefined ? "line-text" : "line-text line-removed"}>{line.text}</p>
            {removed !== undefined && mark === undefined && <p className="hint">Removed at {timeOf(removed)}.</p>}
        </>
    )
}

function partyOf(accountId: string, parties: Parties | undefined): "reporter" | "reported" | undefined {
    if (accountId === "" || parties === undefined) return undefined
    if (accountId === parties.reported.accountId) return "reported"
    if (accountId === parties.reporter.accountId) return "reporter"
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
