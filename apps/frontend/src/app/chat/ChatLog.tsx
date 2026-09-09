import {useEffect, useRef, useState} from "react";
import {ChatMessage} from "../../backends/chat.ts";
import {Countries} from "../../domain/countries.ts";
import {ChevronIcon} from "../components/icons.tsx";
import {truncate} from "../truncate.ts";
import {authorStyle} from "./authorStyle.ts";

export type ChatLogProps = {
    messages: ChatMessage[]
    loading: boolean
    flashing?: ReadonlySet<string>
}

const AUTHOR_MAX_LENGTH = 16

const clock = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})

const PINNED_SLACK_PX = 40

export default function ChatLog(props: ChatLogProps) {
    const scroll = useRef<HTMLDivElement>(null)
    const pinned = useRef(true)
    const [behind, setBehind] = useState(false)

    useEffect(() => {
        const element = scroll.current
        if (!element) return

        // Reading a message further up is not interrupted by a new one landing:
        // the pill says it is there instead of yanking the log down.
        if (!pinned.current) {
            setBehind(true)
            return
        }

        element.scrollTop = element.scrollHeight
    }, [props.messages])

    const onScroll = () => {
        const element = scroll.current
        if (!element) return
        const distance = element.scrollHeight - element.scrollTop - element.clientHeight
        pinned.current = distance <= PINNED_SLACK_PX
        if (pinned.current) setBehind(false)
    }

    const jumpToLatest = () => {
        const element = scroll.current
        if (!element) return
        element.scrollTop = element.scrollHeight
        pinned.current = true
        setBehind(false)
    }

    if (props.loading) {
        return <div className="chat-log chat-log-empty">
            <p>Loading the chat…</p>
        </div>
    }

    if (props.messages.length === 0) {
        return <div className="chat-log chat-log-empty">
            <p>Nobody has said anything yet. Go on.</p>
        </div>
    }

    return <div className="chat-log-shell">
        <div className="chat-log" ref={scroll} onScroll={onScroll}>
            <ul className="chat-messages" aria-live="polite">
                {props.messages.map(message => <li key={message.id}
                                                   className={messageClass(props.flashing, message.id)}
                                                   style={authorStyle(message.authorName, message.authorTag)}>
                    <div className="chat-message-head">
                        <span className="chat-message-country"
                              title={Countries.get(message.countryCode)?.name ?? message.countryCode}>
                            {message.countryCode.toUpperCase()}
                        </span>
                        <span className="chat-message-author">
                            {truncate(message.authorName, AUTHOR_MAX_LENGTH)}
                        </span>
                        <span className="chat-message-tag">#{message.authorTag}</span>
                        <time className="chat-message-time"
                              dateTime={new Date(message.sentAt).toISOString()}>
                            {clock.format(message.sentAt)}
                        </time>
                    </div>
                    <p className="chat-message-text">{message.text}</p>
                </li>)}
            </ul>
        </div>

        {behind && <button type="button" className="chat-jump" onClick={jumpToLatest}>
            New messages
            <ChevronIcon size={14}/>
        </button>}
    </div>
}

function messageClass(flashing: ReadonlySet<string> | undefined, id: string): string {
    return flashing?.has(id) ? "chat-message chat-message-new" : "chat-message"
}
