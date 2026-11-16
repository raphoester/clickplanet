import {useEffect} from "react"
import {create} from "zustand"
import {PlayerLevel} from "../../backends/progression.ts"
import {newerLevel} from "../../domain/level.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useReadsAfterClicks} from "../clicks/useAcceptedClicks.ts"

// The page's own level, on its bar. None for a guest, or before the server keeps one.
type LevelStore = {
    own?: PlayerLevel
    heard: (level: PlayerLevel | undefined) => void
    forget: () => void
}

export const useLevel = create<LevelStore>()((set) => ({
    heard: (level) => set((state) => {
        const own = newerLevel(state.own, level)
        return own === state.own ? state : {own}
    }),
    forget: () => set({own: undefined}),
}))

export type LevelReader = {
    level(): Promise<PlayerLevel | undefined>
}

// Reads the signed-in account's level when it loads and again once its clicks settle. The stream's level_reached
// goes to the same store, from PlayScreen.
export function useLevelFeed(reader: LevelReader, accountId: string | undefined, linked: boolean, listenForClicks: ListenForClicks): void {
    const reads = useReadsAfterClicks(linked ? listenForClicks : undefined)

    useEffect(() => {
        if (!linked) return
        let stale = false
        reader.level().then(
            (level) => {
                if (!stale) useLevel.getState().heard(level)
            },
            (e) => console.error("Could not read the level", e),
        )
        return () => {
            stale = true
        }
    }, [reader, accountId, linked, reads])

    useEffect(() => {
        if (!linked) return
        return () => useLevel.getState().forget()
    }, [accountId, linked])
}
