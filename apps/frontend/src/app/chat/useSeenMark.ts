import {useCallback, useEffect, useRef} from "react"
import {ChatNoSessionError, ChatSeenMarker} from "../../backends/chat.ts"

const SEEN_DELAY_MS = 2_000

export function useSeenMark(marker: ChatSeenMarker | undefined, kept: number): (at: number) => void {
    const sent = useRef(0)
    const wanted = useRef(0)
    const timer = useRef<number>()

    useEffect(() => {
        sent.current = Math.max(sent.current, kept)
    }, [kept])

    const flush = useCallback(() => {
        window.clearTimeout(timer.current)
        timer.current = undefined
        if (!marker || wanted.current <= sent.current) return

        sent.current = wanted.current
        marker.markSeen(wanted.current).catch(e => {
            // No account yet: the page has never minted, so there is nobody to keep it for.
            if (!(e instanceof ChatNoSessionError)) console.error("What the chat has seen could not be kept", e)
        })
    }, [marker])

    useEffect(() => {
        const onPageHide = () => flush()
        window.addEventListener("pagehide", onPageHide)
        return () => {
            window.removeEventListener("pagehide", onPageHide)
            flush()
        }
    }, [flush])

    return useCallback((at: number) => {
        if (at <= wanted.current) return
        wanted.current = at
        timer.current ??= window.setTimeout(flush, SEEN_DELAY_MS)
    }, [flush])
}
