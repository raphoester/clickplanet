import {useCallback, useEffect, useRef, useState} from 'react';
import {
    ChatAnnouncement,
    ChatBackend,
    ChatBlockedError,
    ChatMessage,
    ChatNoSessionError,
    ChatRateLimitedError,
    ChatRejectedError,
    OutgoingMessage,
    OutgoingReaction,
} from '../../backends/chat.ts';
import {addAnnouncements, addMessages, nameSentUnder} from '../../domain/chatLog.ts';
import {
    applyReactionsAnswer,
    applyReactionsChange,
    toggledReactions,
    withReactions,
} from '../../domain/reactions.ts';

export type ChatStatus = 'loading' | 'ready' | 'unavailable'

export type ChatSendFailure = 'rate-limited' | 'blocked' | 'rejected' | 'no-session' | 'failed'

export type UseChatOptions = {
    backend?: ChatBackend
    /**
     * The signed-in player's username, which the server posts and reacts
     * under. A guest has none: the server picks its name, so this hook only
     * learns it once this tab has posted.
     */
    username?: string
}

const NOTHING_SENT: ReadonlySet<string> = new Set()

export function useChat({backend, username}: UseChatOptions) {
    const [messages, setMessages] = useState<ChatMessage[]>([])
    const [announcements, setAnnouncements] = useState<ChatAnnouncement[]>([])
    const [status, setStatus] = useState<ChatStatus>(backend ? 'loading' : 'unavailable')
    const [failure, setFailure] = useState<ChatSendFailure | undefined>(undefined)
    const [mine, setMine] = useState<ReadonlySet<string>>(NOTHING_SENT)

    // What everyone else sees on this player's messages and reactions.
    const displayName = username ?? nameSentUnder(messages, mine)

    // `react` reads the name as it fires rather than closing over it, so a
    // guest learning its name does not build a new callback on every message.
    const named = useRef(displayName)
    named.current = displayName

    const receive = useCallback((incoming: ChatMessage[]) => {
        setMessages(current => addMessages(current, incoming))
    }, [])

    useEffect(() => {
        if (!backend) {
            setStatus('unavailable')
            return
        }

        setStatus('loading')
        setMessages([])
        setAnnouncements([])
        setMine(NOTHING_SENT)

        const abort = new AbortController()
        const stopListening = backend.listenForMessages(
            message => receive([message]),
            change => setMessages(current => applyReactionsChange(current, change)),
            announcement => setAnnouncements(current => addAnnouncements(current, [announcement])),
        )

        backend.getHistory(abort.signal)
            .then(history => {
                if (abort.signal.aborted) return
                receive(history.messages)
                setAnnouncements(current => addAnnouncements(current, history.announcements))
                setStatus('ready')
            })
            .catch(e => {
                if (abort.signal.aborted) return
                console.error("The chat history could not be loaded", e)
                setStatus('unavailable')
            })

        return () => {
            abort.abort()
            stopListening()
        }
    }, [backend, receive])

    const send = useCallback(async (message: OutgoingMessage) => {
        if (!backend) return false

        setFailure(undefined)

        try {
            const sent = await backend.sendMessage(message)
            setMine(current => new Set(current).add(sent.id))
            receive([sent])
            return true
        } catch (e) {
            console.error("The message could not be sent", e)
            setFailure(failureOf(e))
            return false
        }
    }, [backend, receive])

    // Shown at once, then corrected by the server's answer, or undone when it refuses.
    const react = useCallback(async (reaction: OutgoingReaction) => {
        if (!backend) return

        const toggle = (on: boolean) => setMessages(current =>
            withReactions(current, reaction.messageId,
                counts => toggledReactions(counts, reaction.reaction, on, named.current)))

        toggle(reaction.on)
        try {
            const answer = await backend.react(reaction)
            setMessages(current => applyReactionsAnswer(current, answer))
        } catch (e) {
            console.error("The reaction could not be sent", e)
            toggle(!reaction.on)
        }
    }, [backend])

    return {messages, announcements, mine, displayName, status, failure, send, react}
}

function failureOf(e: unknown): ChatSendFailure {
    if (e instanceof ChatRateLimitedError) return 'rate-limited'
    if (e instanceof ChatBlockedError) return 'blocked'
    if (e instanceof ChatRejectedError) return 'rejected'
    if (e instanceof ChatNoSessionError) return 'no-session'
    return 'failed'
}
