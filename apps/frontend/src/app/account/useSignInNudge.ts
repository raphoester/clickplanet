import {useCallback, useEffect, useRef, useState} from "react"
import {quizShown, useQuiz} from "../quiz/quizStore.ts"

// A young account's first dry click brings its welcome quiz within about a second: the nudge leaves it that room.
export const NUDGE_AFTER_MS = 1_500

export type Nudge = "dry" | "slowed"

export function useSignInNudge(offered: boolean, dryTaps: number, fastStart: boolean): {open?: Nudge, close: () => void} {
    const quizUp = useQuiz(quizShown)
    const seenTaps = useRef(dryTaps)
    const wasFast = useRef(fastStart)
    const [due, setDue] = useState<Nudge>()
    const [open, setOpen] = useState<Nudge>()
    const [shown, setShown] = useState<readonly Nudge[]>([])

    useEffect(() => {
        if (dryTaps === seenTaps.current) return
        seenTaps.current = dryTaps
        if (offered && !quizUp && open === undefined && !shown.includes("dry")) setDue("dry")
    }, [dryTaps, offered, quizUp, open, shown])

    useEffect(() => {
        if (fastStart === wasFast.current) return
        wasFast.current = fastStart
        if (!fastStart && offered && open === undefined && !shown.includes("slowed")) setDue("slowed")
    }, [fastStart, offered, open, shown])

    useEffect(() => {
        if (!due) return
        if (!offered || (quizUp && due === "dry")) {
            setDue(undefined)
            return
        }
        if (quizUp) return

        const timer = setTimeout(() => {
            setDue(undefined)
            setOpen(due)
            setShown((before) => [...before, due])
        }, NUDGE_AFTER_MS)
        return () => clearTimeout(timer)
    }, [due, offered, quizUp])

    const close = useCallback(() => setOpen(undefined), [])

    return {open, close}
}
