import {useEffect, useId, useRef, useState} from "react";
import {ChatBackend, OutgoingMessage} from "../../backends/chat.ts";
import {Country} from "../../domain/countries.ts";
import {unreadSince} from "../../domain/chatLog.ts";
import {ChevronIcon} from "../components/icons.tsx";
import {opensFolded} from "../compact.ts";
import ChatComposer from "./ChatComposer.tsx";
import ChatLog from "./ChatLog.tsx";
import {useChat} from "./useChat.ts";
import {useChatIdentity} from "./useChatIdentity.ts";
import "./ChatPanel.css"

export type ChatPanelProps = {
    backend?: ChatBackend
    country: Country
}

const UNREAD_CAP = 99

export default function ChatPanel(props: ChatPanelProps) {
    const [isOpen, setIsOpen] = useState(() => !opensFolded())
    const [unread, setUnread] = useState(0)
    const bodyId = useId()

    const {messages, status, failure, send} = useChat({backend: props.backend})
    const {identity, setName} = useChatIdentity()

    const lastSeen = useRef<string | undefined>(undefined)
    const seenAnything = useRef(false)

    useEffect(() => {
        const last = messages[messages.length - 1]?.id

        if (isOpen || !seenAnything.current) {
            seenAnything.current = seenAnything.current || messages.length > 0
            lastSeen.current = last
            setUnread(0)
            return
        }

        setUnread(unreadSince(messages, lastSeen.current))
    }, [messages, isOpen])

    if (status === 'unavailable') return null

    const onSend = (text: string) => {
        const message: OutgoingMessage = {
            authorName: identity.name,
            authorId: identity.authorId,
            countryCode: props.country.code,
            text,
        }
        return send(message)
    }

    return <section className={isOpen ? "chat chat-open" : "chat"} aria-label="Live chat">
        <button type="button"
                className="chat-header"
                aria-expanded={isOpen}
                aria-controls={bodyId}
                onClick={() => setIsOpen(!isOpen)}>
            <span className="chat-header-title">Chat</span>
            {!isOpen && unread > 0 &&
                <span className="chat-badge"
                      aria-label={unread === 1 ? "1 new message" : `${unread} new messages`}>
                    {unread > UNREAD_CAP ? `${UNREAD_CAP}+` : unread}
                </span>}
            <span className={isOpen ? "chat-chevron chat-chevron-open" : "chat-chevron"}>
                <ChevronIcon/>
            </span>
        </button>

        {isOpen && <div className="chat-body" id={bodyId}>
            <ChatLog messages={messages} loading={status === 'loading'}/>

            <ChatComposer identity={identity}
                          setName={setName}
                          failure={failure}
                          onSend={onSend}/>
        </div>}
    </section>
}
