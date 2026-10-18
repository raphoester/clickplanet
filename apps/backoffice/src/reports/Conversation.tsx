import {useCallback, useState} from "react"
import {ChatAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {clockOf, timeOf} from "../components/time.ts"
import {GetMessageResponse, KeptMessage, KeptQuote} from "../gen/grpc/chat/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"

// How many messages the first page on each side of the reported one holds, and each page asked for after.
const FIRST_PAGE = 10
const NEXT_PAGE = 20

export const CHAT_GONE = "That part of the chat is no longer kept."

// An account a report names, so the conversation marks what it said.
export type Party = {accountId: string, name: string}
export type Parties = {reporter: Party, reported: Party}

// A page is full when it holds as many as asked: there may be more past it.
type Page = {messages: KeptMessage[], more: boolean}
type Cursor = {beforeMessageId: string} | {afterMessageId: string}

async function pageOf(chat: ChatAdmin, cursor: Cursor, limit: number, signal?: AbortSignal): Promise<Page> {
    const answer = await chat.listMessages({...cursor, limit}, {signal})
    return {messages: answer.messages, more: answer.messages.length === limit}
}

type Props = {
    chat: ChatAdmin
    message: GetMessageResponse
    removedAt: Date | undefined
    parties: Parties
}

// The message reported among the pages of the chat just before and just after it, oldest first, the reported one
// shown while they load.
export function Conversation({chat, message, removedAt, parties}: Props) {
    const messageId = message.messageId
    const first = useCallback(
        (signal: AbortSignal) => Promise.all([
            pageOf(chat, {beforeMessageId: messageId}, FIRST_PAGE, signal),
            pageOf(chat, {afterMessageId: messageId}, FIRST_PAGE, signal),
        ]).then(([before, after]) => ({before, after})),
        [chat, messageId],
    )
    const [call, again] = useCall(first)
    const reported = {line: lineOf(message, parties), removedAt}

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

// GetMessage names no author: the report does, for the account it names.
function lineOf(message: GetMessageResponse, parties: Parties): KeptMessage {
    const party = [parties.reported, parties.reporter].find(each => each.accountId === message.accountId)
    return new KeptMessage({
        messageId: message.messageId,
        accountId: message.accountId,
        authorName: message.accountId === "" ? "" : party?.name ?? "",
        text: message.text,
        sentAt: message.sentAt,
        countryId: message.countryId,
        removedAt: message.removedAt,
    })
}

type Reported = {line: KeptMessage, removedAt: Date | undefined}

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
function More({label, load, onLoaded}: {label: string, load: () => Promise<Page>, onLoaded: (page: Page) => void}) {
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
            <Line line={reported.line} parties={parties} day={undefined} removedAt={reported.removedAt} reported/>
            {after.map(line => <Line key={line.messageId} line={line} parties={parties} day={day}/>)}
        </ol>
    )
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
                    <Sent at={line.sentAt?.toDate()} day={day}/>
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
