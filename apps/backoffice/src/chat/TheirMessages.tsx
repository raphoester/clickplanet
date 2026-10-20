import {useCallback, useState} from "react"
import {ChatAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {Page, pageOf, Parties} from "./chatPages.ts"
import {Conversation, LineBody, More} from "./Conversation.tsx"

// How many of their messages each page holds, the newest first.
export const THEIR_PAGE = 20

type Props = {
    chat: ChatAdmin
    accountId: string
    name: string
    parties: Parties | undefined
}

// What one account said in the chat, by its account id: a report from a player's card names no message.
export function TheirMessages({chat, accountId, name, parties}: Props) {
    const first = useCallback(
        (signal: AbortSignal) => pageOf(chat, {accountId}, THEIR_PAGE, signal),
        [chat, accountId],
    )
    const [call, again] = useCall(first)

    return (
        <section className="panel" aria-label="Their messages">
            <h2>{name === "" ? "Their messages" : `Messages from ${name}`}</h2>
            {call.state === "loading" && <p className="muted">Loading…</p>}
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {call.state === "done" && <Pages chat={chat} accountId={accountId} first={call.value} parties={parties}/>}
        </section>
    )
}

type PagesProps = {chat: ChatAdmin, accountId: string, first: Page, parties: Parties | undefined}

function Pages({chat, accountId, first, parties}: PagesProps) {
    const [page, setPage] = useState(first)
    const oldest = page.messages[0]?.messageId ?? ""

    if (page.messages.length === 0) return <p className="muted">Nothing they said in the chat is kept.</p>
    return (
        <>
            <ol className="lines" aria-label="Their messages, newest first">
                {[...page.messages].reverse().map(message => (
                    <Theirs key={message.messageId} chat={chat} message={message} parties={parties}/>
                ))}
            </ol>
            {page.more && (
                <More
                    label="Show earlier"
                    load={() => pageOf(chat, {accountId, beforeMessageId: oldest}, THEIR_PAGE)}
                    onLoaded={older => setPage(shown => ({messages: [...older.messages, ...shown.messages], more: older.more}))}
                />
            )}
        </>
    )
}

function Theirs({chat, message, parties}: {chat: ChatAdmin, message: KeptMessage, parties: Parties | undefined}) {
    const [open, setOpen] = useState(false)

    return (
        <li className="line">
            <LineBody line={message} parties={undefined} day={undefined}/>
            <button type="button" className="button-small" aria-expanded={open} onClick={() => setOpen(shown => !shown)}>
                {open ? "Hide what was said around it" : "Show what was said around it"}
            </button>
            {open && (
                <Conversation chat={chat} anchor={message} removedAt={undefined} parties={parties} mark="This message"/>
            )}
        </li>
    )
}
