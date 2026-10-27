import {useCallback, useMemo, useState} from "react"
import {Race} from "../../backends/standings.ts"
import {Battle} from "../../domain/battle.ts"
import {Reveal, REVEAL_SEEN_STORAGE_KEY, revealKeyOf, revealsDue} from "../../domain/roundReveal.ts"

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
    due: readonly Reveal[]
    dismiss: (reveal: Reveal) => void
}

export function useRoundReveal(race: Race | undefined, battle: Battle | undefined): RoundRevealState {
    const [openedAt] = useState(Date.now)
    const [seen, setSeen] = useState(readSeen)
    const closed = race?.closed

    const dismiss = useCallback((reveal: Reveal) => {
        const key = revealKeyOf(reveal)
        writeSeen(key)
        setSeen(key)
    }, [])

    const due = useMemo(() => revealsDue(closed, openedAt, seen, battle), [closed, openedAt, seen, battle])
    return {due, dismiss}
}
