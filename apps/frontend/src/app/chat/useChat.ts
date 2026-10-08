import {useCallback, useEffect, useRef, useState} from 'react';
import {
    ChatAnnouncement,
    ChatBackend,
    ChatBlockedError,
    ChatMessage,
    ChatMutedError,
    ChatNoSessionError,
    ChatRateLimitedError,
    ChatRejectedError,
    OutgoingMessage,
    OutgoingReaction,
} from '../../backends/chat.ts';
import {addAnnouncements, addMessages, nameSentUnder, newestAt} from '../../domain/chatLog.ts';
import {
    applyReactionsAnswer,
    applyReactionsChange,
    toggledReactions,
    withNewerReactions,
    withReactions,
} from '../../domain/reactions.ts';

export type ChatStatus = 'loading' | 'ready' | 'unavailable'

export type ChatSendFailure = 'rate-limited' | 'blocked' | 'muted' | 'rejected' | 'no-session' | 'failed'

export type SeenAtLoad = {
    until: number
    kept: boolean
}

export type UseChatOptions = {
    backend?: ChatBackend
    username?: string
}

const NOTHING_SENT: ReadonlySet<string> = new Set()

export function useChat({backend, username}: UseChatOptions) {
    const [messages, setMessages] = useState<ChatMessage[]>([])
    const [announcements, setAnnouncements] = useState<ChatAnnouncement[]>([])
    const [status, setStatus] = useState<ChatStatus>(backend ? 'loading' : 'unavailable')
    const [failure, setFailure] = useState<ChatSendFailure | undefined>(undefined)
    const [mutedUntil, setMutedUntil] = useState<number>()
    const [mine, setMine] = useState<ReadonlySet<string>>(NOTHING_SENT)
    const [seenAtLoad, setSeenAtLoad] = useState<SeenAtLoad>()

    const displayName = username ?? nameSentUnder(messages, mine)

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
        setSeenAtLoad(undefined)

        const abort = new AbortController()

        const catchUp = () => backend.getHistory(abort.signal)
            .then(history => {
                if (abort.signal.aborted) return
                setMessages(current => withNewerReactions(addMessages(current, history.messages), history.messages))
                setAnnouncements(current => addAnnouncements(current, history.announcements))
            })
            .catch(e => {
                if (abort.signal.aborted) return
                console.error("The chat could not catch up", e)
            })

        const stopListening = backend.listenForMessages(
            message => receive([message]),
            change => setMessages(current => applyReactionsChange(current, change)),
            announcement => setAnnouncements(current => addAnnouncements(current, [announcement])),
            () => void catchUp(),
        )

        backend.getHistory(abort.signal)
            .then(history => {
                if (abort.signal.aborted) return
                receive(history.messages)
                setAnnouncements(current => addAnnouncements(current, history.announcements))
                setSeenAtLoad(history.seenUntil !== undefined
                    ? {until: history.seenUntil, kept: true}
                    : {until: newestAt(history.messages, history.announcements) ?? 0, kept: false})
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
            if (e instanceof ChatMutedError) setMutedUntil(e.until)
            setFailure(failureOf(e))
            return false
        }
    }, [backend, receive])

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
            if (e instanceof ChatMutedError) {
                setMutedUntil(e.until)
                setFailure('muted')
            }
        }
    }, [backend])

    return {messages, announcements, mine, displayName, seenAtLoad, status, failure, mutedUntil, send, react}
}

function failureOf(e: unknown): ChatSendFailure {
    if (e instanceof ChatRateLimitedError) return 'rate-limited'
    if (e instanceof ChatMutedError) return 'muted'
    if (e instanceof ChatBlockedError) return 'blocked'
    if (e instanceof ChatRejectedError) return 'rejected'
    if (e instanceof ChatNoSessionError) return 'no-session'
    return 'failed'
}
