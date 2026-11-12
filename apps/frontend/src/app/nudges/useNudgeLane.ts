import {useEffect, useState} from "react"
import {NUDGE_SETTLE_MS} from "../../domain/nudges.ts"

// Open once the map showed for a while, so the moments due at the start are queued before any nudge.
export function useNudgeLane(shown: boolean): boolean {
    const [open, setOpen] = useState(false)

    useEffect(() => {
        if (!shown || open) return
        const timer = setTimeout(() => setOpen(true), NUDGE_SETTLE_MS)
        return () => clearTimeout(timer)
    }, [shown, open])

    return open
}
