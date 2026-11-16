import {useEffect, useState} from "react"
import {create} from "zustand"
import {SessionSource} from "../../backends/backend.ts"
import {MissionRates} from "../../backends/missions.ts"
import {LevelRead, LevelReader, PlayerLevel} from "../../backends/progression.ts"
import {UnlockLevels} from "../../domain/bonus.ts"
import {newerLevel} from "../../domain/level.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useReadsAfterClicks} from "../clicks/useAcceptedClicks.ts"

// What GetLevel answered, for every part of the page that shows it: the dock's bar, the Progress tab, the profile and
// the bonus bar's locks and the XP of each mission. `own` is none for a guest, or before the server keeps a level.
type LevelStore = {
    own?: PlayerLevel
    unlocks: UnlockLevels
    missionRates: MissionRates
    read: (read: LevelRead) => void
    reached: (level: PlayerLevel) => void
    forget: () => void
}

const NO_UNLOCKS: UnlockLevels = {}
const NO_RATES: MissionRates = {}

export const useLevel = create<LevelStore>()((set) => ({
    unlocks: NO_UNLOCKS,
    missionRates: NO_RATES,
    read: (read) => set((state) => ({
        own: newerLevel(state.own, read.level), unlocks: read.unlocks, missionRates: read.missionRates,
    })),
    reached: (level) => set((state) => {
        const own = newerLevel(state.own, level)
        return own === state.own ? state : {own}
    }),
    forget: () => set({own: undefined, unlocks: NO_UNLOCKS, missionRates: NO_RATES}),
}))

export type LevelCaller = {
    accountId: string | undefined
    linked: boolean
    listenForClicks: ListenForClicks
}

// One GetLevel read at load, on each new session (a guest's first click makes one) or account, and, signed in, once its
// clicks settle. The stream's level_reached goes to the same store, from PlayScreen.
export function useLevelFeed(reader: LevelReader, sessions: SessionSource, {accountId, linked, listenForClicks}: LevelCaller): void {
    const reads = useReadsAfterClicks(linked ? listenForClicks : undefined)
    const [session, setSession] = useState<string>()

    useEffect(() => sessions.listenForSessions(setSession), [sessions])

    useEffect(() => {
        let stale = false
        reader.readLevel().then(
            (read) => {
                if (!stale && read) useLevel.getState().read(read)
            },
            (e) => console.error("Could not read the level", e),
        )
        return () => {
            stale = true
        }
    }, [reader, session, accountId, linked, reads])

    useEffect(() => () => useLevel.getState().forget(), [accountId, linked])
}
