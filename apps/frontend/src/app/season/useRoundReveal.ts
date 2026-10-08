import {useCallback, useState} from "react"
import {ClosedRound, Race} from "../../backends/standings.ts"
import {REVEAL_SEEN_STORAGE_KEY, revealDue, revealKeyOf} from "../../domain/roundReveal.ts"

function readSeen(): string | null {
    try {
        return window.localStorage.getItem(REVEAL_SEEN_STORAGE_KEY)
    } catch {
        return null
    }
}

function writeSeen(key: string) {
    try {
        window.localStorage.setItem(REVEAL_SEEN_STORAGE_KEY, key)
    } catch (e) {
        console.error("Could not remember the round shown", e)
    }
}

export type RoundRevealState = {
    closed?: ClosedRound
    dismiss: () => void
}

export function useRoundReveal(race: Race | undefined): RoundRevealState {
    const [openedAt] = useState(Date.now)
    const [seen, setSeen] = useState(readSeen)
    const closed = race?.closed

    const dismiss = useCallback(() => {
        if (!closed) return
        const key = revealKeyOf(closed)
        writeSeen(key)
        setSeen(key)
    }, [closed])

    return {closed: revealDue(closed, openedAt, seen) ? closed : undefined, dismiss}
}
