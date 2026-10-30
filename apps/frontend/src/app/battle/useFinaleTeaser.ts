import {useCallback, useState} from "react"
import {Finale, FINALE_TEASED_STORAGE_KEY, finaleTeaserDue} from "../../domain/battle.ts"

function readSeen(): string | null {
    try {
        return window.localStorage.getItem(FINALE_TEASED_STORAGE_KEY)
    } catch {
        return null
    }
}

function writeSeen(roundId: string) {
    try {
        window.localStorage.setItem(FINALE_TEASED_STORAGE_KEY, roundId)
    } catch (e) {
        console.error("Could not remember the Final Battle teased", e)
    }
}

export type FinaleTeaserState = {
    finale?: Finale
    dismiss: (finale: Finale) => void
}

export function useFinaleTeaser(finale: Finale | undefined, now: number): FinaleTeaserState {
    const [seen, setSeen] = useState(readSeen)

    const dismiss = useCallback((teased: Finale) => {
        writeSeen(teased.roundId)
        setSeen(teased.roundId)
    }, [])

    return {finale: finaleTeaserDue(finale, seen, now) ? finale : undefined, dismiss}
}
