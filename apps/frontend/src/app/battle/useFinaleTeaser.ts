import {useCallback, useEffect, useState} from "react"
import {Finale, FINALE_TEASED_STORAGE_KEY, FINALE_TEASER_AFTER_CLICKS, finaleTeaserDue} from "../../domain/battle.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"

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

export function useFinaleTeaser(finale: Finale | undefined, now: number, listenForClicks: ListenForClicks): FinaleTeaserState {
    const [seen, setSeen] = useState(readSeen)
    const [clicks, setClicks] = useState(0)
    const played = clicks >= FINALE_TEASER_AFTER_CLICKS

    useEffect(() => {
        if (played) return
        return listenForClicks(() => setClicks((count) => count + 1))
    }, [listenForClicks, played])

    const dismiss = useCallback((teased: Finale) => {
        writeSeen(teased.roundId)
        setSeen(teased.roundId)
    }, [])

    return {finale: finaleTeaserDue(finale, seen, now, clicks) ? finale : undefined, dismiss}
}
