import {useCallback, useEffect, useState} from 'react';
import {
    ChatBackend,
    ChatBlockedError,
    ChatMessage,
    ChatRateLimitedError,
    ChatRejectedError,
    ChatUnavailableError,
    OutgoingMessage,
} from '../../backends/chat.ts';
import {addMessages} from '../../domain/chatLog.ts';

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
        const stopListening = backend.listenForMessages(message => receive([message]))

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
            if (e instanceof ChatUnavailableError) setStatus('unavailable')
            setFailure(failureOf(e))
            return false
        }
    }, [backend, receive])

    return {messages, mine, status, failure, send}
}

function failureOf(e: unknown): ChatSendFailure {
    if (e instanceof ChatRateLimitedError) return 'rate-limited'
    if (e instanceof ChatBlockedError) return 'blocked'
    if (e instanceof ChatRejectedError) return 'rejected'
    return 'failed'
}
