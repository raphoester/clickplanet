import {useEffect, useRef, useState} from "react";
import {ChatMessage} from "../../backends/chat.ts";
import {Countries, nameWithoutFlag} from "../../domain/countries.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import {ChevronIcon} from "../components/icons.tsx";
import {startsGroup} from "../../domain/chatLog.ts";
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
                {props.messages.map((message, index) => {
                    const opens = startsGroup(props.messages[index - 1], message)

                    return <li key={message.id}
                               className={messageClass(props.flashing, message.id, opens)}
                               style={authorStyle(message.authorName, message.authorTag)}>
                        {opens && <div className="chat-message-head">
                            <span className="chat-message-country"
                                  role="img"
                                  aria-label={countryName(message.countryCode)}
                                  title={countryName(message.countryCode)}>
                                <CountryFlag code={message.countryCode}/>
                            </span>
                            <span className="chat-message-author">
                                {truncate(message.authorName, AUTHOR_MAX_LENGTH)}
                            </span>
                            <span className="chat-message-tag">#{message.authorTag}</span>
                            <time className="chat-message-time"
                                  dateTime={new Date(message.sentAt).toISOString()}>
                                {clock.format(message.sentAt)}
                            </time>
                        </div>}

                        <p className="chat-message-text">{message.text}</p>
                    </li>
                })}
            </ul>
        </div>

        {behind && <button type="button" className="chat-jump" onClick={jumpToLatest}>
            New messages
            <ChevronIcon size={14}/>
        </button>}
    </div>
}

function messageClass(
    flashing: ReadonlySet<string> | undefined,
    id: string,
    opens: boolean,
): string {
    return [
        "chat-message",
        opens ? "chat-message-opens" : "chat-message-cont",
        flashing?.has(id) && "chat-message-new",
    ].filter(Boolean).join(" ")
}

// The badge is a flag and nothing else, so the name it stands for is what the
// tooltip and the accessibility tree carry.
function countryName(code: string): string {
    const country = Countries.get(code)
    return country ? nameWithoutFlag(country) : code
}
