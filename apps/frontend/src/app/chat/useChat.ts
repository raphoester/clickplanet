import {useCallback, useEffect, useState} from 'react';
import {
    ChatBackend,
    ChatBlockedError,
    ChatMessage,
    ChatRateLimitedError,
    ChatRejectedError,
    OutgoingMessage,
    OutgoingReaction,
} from '../../backends/chat.ts';
import {addMessages} from '../../domain/chatLog.ts';
import {applyReactionsChange, toggledReactions, withReactions} from '../../domain/reactions.ts';

export type ChatStatus = 'loading' | 'ready' | 'unavailable'

export type ChatSendFailure = 'rate-limited' | 'blocked' | 'rejected' | 'failed'

export type UseChatOptions = {
    backend?: ChatBackend
}

const NOTHING_SENT: ReadonlySet<string> = new Set()

export function useChat({backend}: UseChatOptions) {
    const [messages, setMessages] = useState<ChatMessage[]>([])
    const [status, setStatus] = useState<ChatStatus>(backend ? 'loading' : 'unavailable')
    const [failure, setFailure] = useState<ChatSendFailure | undefined>(undefined)
    const [mine, setMine] = useState<ReadonlySet<string>>(NOTHING_SENT)

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
        setMine(NOTHING_SENT)

        const abort = new AbortController()
        const stopListening = backend.listenForMessages(
            message => receive([message]),
            change => setMessages(current => applyReactionsChange(current, change)),
        )

        backend.getHistory(abort.signal)
            .then(history => {
                if (abort.signal.aborted) return
                receive(history)
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
            withReactions(current, reaction.messageId, counts => toggledReactions(counts, reaction.reaction, on)))

        toggle(reaction.on)
        try {
            const counts = await backend.react(reaction)
            setMessages(current => withReactions(current, reaction.messageId, () => counts))
        } catch (e) {
            console.error("The reaction could not be sent", e)
            toggle(!reaction.on)
        }
    }, [backend])

    return {messages, mine, status, failure, send, react}
}

function failureOf(e: unknown): ChatSendFailure {
    if (e instanceof ChatRateLimitedError) return 'rate-limited'
    if (e instanceof ChatBlockedError) return 'blocked'
    if (e instanceof ChatRejectedError) return 'rejected'
    return 'failed'
}
