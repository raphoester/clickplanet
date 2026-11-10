import {useCallback, useEffect, useRef, useState} from 'react';
import {
    ChatAnnouncement,
    ChatBackend,
    ChatBlockedError,
    ChatMessage,
    ChatMessageGoneError,
    ChatMutedError,
    ChatNoSessionError,
    ChatNotInRoomError,
    ChatRateLimitedError,
    ChatRejectedError,
    OutgoingMessage,
    OutgoingReaction,
} from '../../backends/chat.ts';
import {accountSentUnder, addAnnouncements, addMessages, newestAt, withRemoved} from '../../domain/chatLog.ts';
import {useNameplates} from '../nameplates/nameplatesStore.ts';
import {
    applyReactionsAnswer,
    applyReactionsChange,
    toggledReactions,
    withNewerReactions,
    withReactions,
} from '../../domain/reactions.ts';

export type ChatStatus = 'loading' | 'ready' | 'unavailable'

export type ChatSendFailure =
    'rate-limited' | 'blocked' | 'muted' | 'gifs-muted' | 'rejected' | 'gone' | 'no-session' | 'failed'

export type SeenAtLoad = {
    until: number
    kept: boolean
}

export type UseRoomLogOptions = {
    backend?: ChatBackend
    accountId?: string
    onLeft?: () => void
}

const NOTHING_SENT: ReadonlySet<string> = new Set()

export function useRoomLog({backend, accountId, onLeft}: UseRoomLogOptions) {
    const [messages, setMessages] = useState<ChatMessage[]>([])
    const [announcements, setAnnouncements] = useState<ChatAnnouncement[]>([])
    const [status, setStatus] = useState<ChatStatus>(backend ? 'loading' : 'unavailable')
    const [failure, setFailure] = useState<ChatSendFailure | undefined>(undefined)
    const [mutedUntil, setMutedUntil] = useState<number>()
    const [mine, setMine] = useState<ReadonlySet<string>>(NOTHING_SENT)
    const [seenAtLoad, setSeenAtLoad] = useState<SeenAtLoad>()

    const ownAccount = accountId ?? accountSentUnder(messages, mine)
    const own = useCallback((message: ChatMessage) =>
        mine.has(message.id) || (ownAccount !== undefined && message.authorAccountId === ownAccount), [mine, ownAccount])

    const me = useRef(ownAccount)
    me.current = ownAccount

    const leaving = useRef(onLeft)
    leaving.current = onLeft
    const leave = useCallback(() => {
        setStatus('unavailable')
        leaving.current?.()
    }, [])

    const room = useRef<AbortSignal>()

    // A message shows once its names are read, so none shows a blank name; a read that hangs holds it 2 s at most.
    const receive = useCallback((incoming: ChatMessage[], signal: AbortSignal | undefined) =>
        useNameplates.getState().whenRead(authorsOf(incoming)).then(() => {
            if (signal?.aborted) return
            setMessages(current => addMessages(current, incoming))
        }), [])

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
        room.current = abort.signal

        const catchUp = () => backend.getHistory(abort.signal)
            .then(async history => {
                if (abort.signal.aborted) return
                await useNameplates.getState().whenRead(authorsOf(history.messages))
                if (abort.signal.aborted) return
                setMessages(current => withNewerReactions(addMessages(current, history.messages), history.messages))
                setAnnouncements(current => addAnnouncements(current, history.announcements))
            })
            .catch(e => {
                if (abort.signal.aborted) return
                console.error("The chat could not catch up", e)
            })

        const stopListening = backend.listenForMessages(
            message => void receive([message], abort.signal),
            change => setMessages(current => applyReactionsChange(current, change)),
            announcement => setAnnouncements(current => addAnnouncements(current, [announcement])),
            () => void catchUp(),
            messageId => setMessages(current => withRemoved(current, messageId)),
            () => {
                if (!abort.signal.aborted) leave()
            },
        )

        backend.getHistory(abort.signal)
            .then(async history => {
                if (abort.signal.aborted) return
                await receive(history.messages, abort.signal)
                if (abort.signal.aborted) return
                setAnnouncements(current => addAnnouncements(current, history.announcements))
                setSeenAtLoad(history.seenUntil !== undefined
                    ? {until: history.seenUntil, kept: true}
                    : {until: newestAt(history.messages, history.announcements) ?? 0, kept: false})
                setStatus('ready')
            })
            .catch(e => {
                if (abort.signal.aborted) return
                if (e instanceof ChatNotInRoomError) {
                    leave()
                    return
                }
                console.error("The chat history could not be loaded", e)
                setStatus('unavailable')
            })

        return () => {
            abort.abort()
            stopListening()
        }
    }, [backend, receive, leave])

    const send = useCallback(async (message: OutgoingMessage): Promise<ChatSendFailure | undefined> => {
        if (!backend) return 'failed'

        setFailure(undefined)
        const signal = room.current

        try {
            const sent = await backend.sendMessage(message)
            if (signal?.aborted) return undefined
            setMine(current => new Set(current).add(sent.id))
            void receive([sent], signal)
            return undefined
        } catch (e) {
            console.error("The message could not be sent", e)
            if (e instanceof ChatMutedError) setMutedUntil(e.until)
            if (e instanceof ChatNotInRoomError) leave()
            const failed = failureOf(e)
            setFailure(failed)
            return failed
        }
    }, [backend, receive, leave])

    const react = useCallback(async (reaction: OutgoingReaction) => {
        if (!backend) return

        const toggle = (on: boolean) => setMessages(current =>
            withReactions(current, reaction.messageId,
                counts => toggledReactions(counts, reaction.reaction, on, me.current)))

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
            if (e instanceof ChatNotInRoomError) leave()
        }
    }, [backend, leave])

    return {messages, announcements, mine, own, ownAccount, seenAtLoad, status, failure, mutedUntil, send, react}
}

function authorsOf(messages: readonly ChatMessage[]): string[] {
    return messages.flatMap(message => [message.authorAccountId, message.replyTo?.authorAccountId ?? ""])
}

function failureOf(e: unknown): ChatSendFailure {
    if (e instanceof ChatRateLimitedError) return 'rate-limited'
    if (e instanceof ChatMutedError) return e.kind === 'gifs' ? 'gifs-muted' : 'muted'
    if (e instanceof ChatBlockedError) return 'blocked'
    if (e instanceof ChatRejectedError) return 'rejected'
    if (e instanceof ChatMessageGoneError) return 'gone'
    if (e instanceof ChatNoSessionError) return 'no-session'
    return 'failed'
}
