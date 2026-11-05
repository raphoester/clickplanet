import {useCallback, useState} from "react"
import {ChatAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {Page, pageOf, Parties} from "./chatPages.ts"
import {CHAT_GONE, LineBody, More} from "./Conversation.tsx"
import {CONVERSATIONS, moreConversations, Read, Window} from "./theirConversations.ts"

// How many of their messages are read at a time, to build conversations around.
export const THEIR_PAGE = 20
// How many messages "Show earlier" and "Show later" add to one conversation.
const NEXT_PAGE = 10

type Props = {
    chat: ChatAdmin
    accountId: string
    parties: Parties | undefined
}

type Built = {
    shown: Window[]
    // Their messages read and in no conversation yet, newest first.
    left: KeptMessage[]
    // Whether more of their messages are kept before the oldest read, and its id.
    theirsMore: boolean
    oldest: string
}

async function theirPage(chat: ChatAdmin, accountId: string, before: string, signal?: AbortSignal) {
    const page = await pageOf(chat, before === "" ? {accountId} : {accountId, beforeMessageId: before}, THEIR_PAGE, signal)
    return {newestFirst: [...page.messages].reverse(), more: page.more, oldest: page.messages[0]?.messageId ?? before}
}

// What one account said in the chat, in the conversations it said it in, by its account id: a report from a
// player's card names no message.
export function TheirMessages({chat, accountId, parties}: Props) {
    const read = useCallback<Read>((ask, limit) => pageOf(chat, ask, limit), [chat])
    const first = useCallback(async (signal: AbortSignal): Promise<Built> => {
        const theirs = await theirPage(chat, accountId, "", signal)
        const built = await moreConversations([], theirs.newestFirst, read, CONVERSATIONS)
        return {...built, theirsMore: theirs.more, oldest: theirs.oldest}
    }, [chat, accountId, read])
    const [call, again] = useCall(first)

    return (
        <>
            {call.state === "loading" && <p className="muted">Loading…</p>}
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {call.state === "done" && (
                <Conversations chat={chat} accountId={accountId} first={call.value} read={read} parties={parties}/>
            )}
        </>
    )
}

type ConversationsProps = {chat: ChatAdmin, accountId: string, first: Built, read: Read, parties: Parties | undefined}

function Conversations({chat, accountId, first, read, parties}: ConversationsProps) {
    const [built, setBuilt] = useState(first)

    if (built.shown.length === 0) return <p className="muted">Nothing they said in the chat is kept.</p>

    const more = async (): Promise<Built> => {
        let {left, theirsMore, oldest} = built
        if (left.length === 0 && theirsMore) {
            const theirs = await theirPage(chat, accountId, oldest)
            left = theirs.newestFirst
            theirsMore = theirs.more
            oldest = theirs.oldest
        }
        const next = await moreConversations(built.shown, left, read, CONVERSATIONS)
        return {...next, theirsMore, oldest}
    }
    const grown = (at: number) => (window: Window) =>
        setBuilt(shown => ({...shown, shown: shown.shown.map((each, i) => i === at ? window : each)}))

    return (
        <>
            <ol className="conversations" aria-label="Their conversations, newest first">
                {built.shown.map((window, at) => (
                    <ConversationWindow
                        key={at}
                        chat={chat}
                        window={window}
                        accountId={accountId}
                        parties={parties}
                        onGrown={grown(at)}
                    />
                ))}
            </ol>
            {(built.left.length > 0 || built.theirsMore) && <BuildMore load={more} onLoaded={setBuilt}/>}
        </>
    )
}

type WindowProps = {
    chat: ChatAdmin
    window: Window
    accountId: string
    parties: Parties | undefined
    onGrown: (window: Window) => void
}

function ConversationWindow({chat, window, accountId, parties, onGrown}: WindowProps) {
    const oldest = window.lines[0]?.messageId ?? ""
    const newest = window.lines.at(-1)?.messageId ?? ""
    const earlier = (page: Page) => onGrown({...window, lines: [...page.messages, ...window.lines], moreBefore: page.more})
    const later = (page: Page) => onGrown({...window, lines: [...window.lines, ...page.messages], moreAfter: page.more})

    return (
        <li className="conversation-window">
            {window.moreBefore && (
                <More label="Show earlier" load={() => pageOf(chat, {beforeMessageId: oldest}, NEXT_PAGE)} onLoaded={earlier}/>
            )}
            <ol className="lines" aria-label="A conversation">
                {window.lines.map(line => (
                    <li key={line.messageId} className={line.accountId === accountId ? "line line-theirs" : "line"}>
                        <LineBody line={line} parties={parties} day={undefined}/>
                    </li>
                ))}
            </ol>
            {window.moreAfter && (
                <More label="Show later" load={() => pageOf(chat, {afterMessageId: newest}, NEXT_PAGE)} onLoaded={later}/>
            )}
        </li>
    )
}

type BuildStep = {step: "idle"} | {step: "running"} | {step: "refused", words: string}

// The next conversations, on a click: reads, so it asks nothing first.
function BuildMore({load, onLoaded}: {load: () => Promise<Built>, onLoaded: (built: Built) => void}) {
    const [step, setStep] = useState<BuildStep>({step: "idle"})

    const run = () => {
        setStep({step: "running"})
        load().then(
            built => {
                setStep({step: "idle"})
                onLoaded(built)
            },
            (err: unknown) => setStep({step: "refused", words: wordsOf(err, {notFound: CHAT_GONE})}),
        )
    }

    return (
        <div className="more">
            <button type="button" className="button-small" disabled={step.step === "running"} onClick={run}>
                {step.step === "running" ? "Loading…" : "Show more conversations"}
            </button>
            {step.step === "refused" && <p className="action-refused" role="alert">{step.words}</p>}
        </div>
    )
}
