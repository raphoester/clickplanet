import {useCallback, useEffect, useRef, useState} from "react";
import {ChatMessage} from "../../backends/chat.ts";
import {idsSince, unseenAfter} from "../../domain/chatLog.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import {useRoomLog, UseRoomLogOptions} from "./useRoomLog.ts";
import {useSeenMark} from "./useSeenMark.ts";

const FLASH_MS = 1600

const TOAST_MS = 4000

const NOTHING: ReadonlySet<string> = new Set()

export type UseChatRoomOptions = UseRoomLogOptions & {
    seeing: boolean
    playSound?: PlaySound
}

export type ChatRoomState = ReturnType<typeof useChatRoom>

export function useChatRoom({seeing, playSound, ...options}: UseChatRoomOptions) {
    const chat = useRoomLog(options)
    const {messages, announcements, mine, own, seenAtLoad, status} = chat
    const markSeen = useSeenMark(options.backend, seenAtLoad?.kept ? seenAtLoad.until : 0)
    const [seenLater, setSeenLater] = useState(0)
    const seenUntil = seenAtLoad === undefined ? undefined : Math.max(seenAtLoad.until, seenLater)
    const [flashing, setFlashing] = useState<ReadonlySet<string>>(NOTHING)
    const [toast, setToast] = useState<ChatMessage>()

    const lastShown = useRef<string | undefined>(undefined)
    const shownAnything = useRef(false)
    const newestId = useRef<string | undefined>(undefined)
    const fading = useRef<number[]>([])
    const lastHeard = useRef<string | undefined>(undefined)
    const heardAnything = useRef(false)

    const flash = useCallback((ids: string[]) => {
        if (ids.length === 0) return

        setFlashing(current => new Set([...current, ...ids]))
        fading.current.push(window.setTimeout(() => setFlashing(current => {
            const rest = new Set(current)
            ids.forEach(id => rest.delete(id))
            return rest
        }), FLASH_MS))
    }, [])

    useEffect(() => () => fading.current.forEach(clearTimeout), [])

    useEffect(() => {
        if (status !== 'loading') return
        setSeenLater(0)
        setToast(undefined)
    }, [status])

    useEffect(() => {
        // So the next visit has a mark to count from, even if the chat is never opened in this one.
        if (seenAtLoad && !seenAtLoad.kept) markSeen(seenAtLoad.until)
    }, [seenAtLoad, markSeen])

    const onSeen = useCallback((at: number) => {
        setSeenLater(current => Math.max(current, at))
        markSeen(at)
    }, [markSeen])

    const unread = seenUntil === undefined || status !== 'ready' ? 0 : unseenAfter(messages, announcements, seenUntil, own)

    useEffect(() => {
        const last = messages[messages.length - 1]

        if (status !== 'ready' || !shownAnything.current) {
            shownAnything.current = status === 'ready' && messages.length > 0
            lastShown.current = last?.id
            return
        }
        if (!seeing) return

        flash(idsSince(messages, lastShown.current).filter(id => !mine.has(id)))
        lastShown.current = last?.id
    }, [status, messages, seeing, mine, flash])

    useEffect(() => {
        if (seeing) setToast(undefined)
    }, [seeing])

    useEffect(() => {
        const last = messages[messages.length - 1]
        if (seenUntil === undefined || !last || last.id === newestId.current) return
        newestId.current = last.id

        if (seeing || last.sentAt <= seenUntil || own(last)) return
        setToast(last)
    }, [seenUntil, messages, seeing, own])

    useEffect(() => {
        if (!toast) return
        const timer = setTimeout(() => setToast(undefined), TOAST_MS)
        return () => clearTimeout(timer)
    }, [toast])

    useEffect(() => {
        const last = messages[messages.length - 1]?.id

        if (status !== 'ready' || !heardAnything.current) {
            heardAnything.current = status === 'ready' && messages.length > 0
            lastHeard.current = last
            return
        }

        const fresh = messages.slice(messages.length - idsSince(messages, lastHeard.current).length)
        lastHeard.current = last

        // By account too: an own message's broadcast can land before `mine` holds its id.
        if (fresh.some(message => !own(message))) playSound?.('chat')
    }, [status, messages, own, playSound])

    return {...chat, seenUntil, onSeen, unread, flashing, toast}
}
