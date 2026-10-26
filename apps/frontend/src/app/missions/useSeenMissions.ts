import {useCallback, useEffect, useState} from "react"
import {Mission} from "../../backends/missions.ts"
import {parseSeenMissions, SEEN_MISSIONS_STORAGE_KEY, SeenMissions, seenNow} from "../../domain/missions.ts"

function readStoredSeen(): string | null {
    try {
        return window.localStorage.getItem(SEEN_MISSIONS_STORAGE_KEY)
    } catch {
        return null
    }
}

export function useSeenMissions() {
    const [seen, setSeen] = useState<SeenMissions>(() => parseSeenMissions(readStoredSeen()))

    useEffect(() => {
        try {
            window.localStorage.setItem(SEEN_MISSIONS_STORAGE_KEY, JSON.stringify(seen))
        } catch (e) {
            console.error("Could not keep which missions were seen", e)
        }
    }, [seen])

    const markSeen = useCallback((missions: readonly Mission[]) => setSeen((was) => {
        const now = seenNow(missions)
        return JSON.stringify(now) === JSON.stringify(was) ? was : now
    }), [])

    return {seen, markSeen}
}
