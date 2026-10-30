import {useCallback, useEffect, useRef, useState} from "react"

// A young account's first dry click brings its welcome quiz within about a second: the nudge leaves it that room.
export const NUDGE_AFTER_MS = 1_500

export function useSignInNudge(offered: boolean, dryTaps: number, quizUp: boolean): {open: boolean, close: () => void} {
    const seen = useRef(dryTaps)
    const [due, setDue] = useState(false)
    const [shown, setShown] = useState<"not yet" | "open" | "closed">("not yet")

    useEffect(() => {
        if (dryTaps === seen.current) return
        seen.current = dryTaps
        if (offered && !quizUp && shown === "not yet") setDue(true)
    }, [dryTaps, offered, quizUp, shown])

    useEffect(() => {
        if (!due) return
        if (!offered || quizUp) {
            setDue(false)
            return
        }

        const timer = setTimeout(() => {
            setDue(false)
            setShown("open")
        }, NUDGE_AFTER_MS)
        return () => clearTimeout(timer)
    }, [due, offered, quizUp])

    const close = useCallback(() => setShown("closed"), [])

    return {open: shown === "open", close}
}
