import {create} from "zustand"
import {Mission} from "../../backends/missions.ts"
import {parseSeenMissions, SEEN_MISSIONS_STORAGE_KEY, SeenMissions, seenNow} from "../../domain/missions.ts"

type SeenMissionsStore = {
    seen: SeenMissions
    markSeen: (missions: readonly Mission[]) => void
}

function readStoredSeen(): string | null {
    try {
        return window.localStorage.getItem(SEEN_MISSIONS_STORAGE_KEY)
    } catch {
        return null
    }
}

export const useSeenMissions = create<SeenMissionsStore>()((set) => ({
    seen: parseSeenMissions(readStoredSeen()),
    markSeen: (missions) => set((state) => {
        const now = seenNow(missions)
        return JSON.stringify(now) === JSON.stringify(state.seen) ? state : {seen: now}
    }),
}))

useSeenMissions.subscribe(({seen}) => {
    try {
        window.localStorage.setItem(SEEN_MISSIONS_STORAGE_KEY, JSON.stringify(seen))
    } catch (e) {
        console.error("Could not keep which missions were seen", e)
    }
})
