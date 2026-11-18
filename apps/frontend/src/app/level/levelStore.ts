import {useEffect, useState} from "react"
import {create} from "zustand"
import {SessionSource} from "../../backends/backend.ts"
import {MissionRates} from "../../backends/missions.ts"
import {LevelRead, LevelReader, PlayerLevel, XpGained, XpRates} from "../../backends/progression.ts"
import {UnlockLevels} from "../../domain/bonus.ts"
import {gainedOn, newerLevel} from "../../domain/level.ts"
import {holdOf, NO_POPS, poppedIn, poppedOut, XpPops} from "../../domain/xpPops.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useReadsAfterClicks} from "../clicks/useAcceptedClicks.ts"

// What GetLevel answered, for every part of the page that shows it: the dock's bar, the Progress tab, the profile and
// the bonus bar's locks and the XP of each mission. `own` is none for a guest, or before the server keeps a level.
// `pops` are the XP gains the stream told, shown one at a time on the dock's line.
type LevelStore = {
    own?: PlayerLevel
    unlocks: UnlockLevels
    missionRates: MissionRates
    xpRates: XpRates
    pops: XpPops
    read: (read: LevelRead) => void
    reached: (level: PlayerLevel) => void
    gained: (gained: XpGained) => void
    popOut: () => void
    forget: () => void
}

const NO_UNLOCKS: UnlockLevels = {}
const NO_RATES: MissionRates = {}
const NO_XP_RATES: XpRates = {}

// The pop shown goes on its own clock, whoever draws it: a gain told while a sheet covers the dock is not shown late.
let popClock: ReturnType<typeof setTimeout> | undefined

function timed(pops: XpPops): XpPops {
    clearTimeout(popClock)
    popClock = pops.shown ? setTimeout(() => useLevel.getState().popOut(), holdOf(pops.shown)) : undefined
    return pops
}

export const useLevel = create<LevelStore>()((set) => ({
    unlocks: NO_UNLOCKS,
    missionRates: NO_RATES,
    xpRates: NO_XP_RATES,
    pops: NO_POPS,
    read: (read) => set((state) => ({
        own: newerLevel(state.own, read.level), unlocks: read.unlocks, missionRates: read.missionRates,
        xpRates: read.xpRates,
    })),
    reached: (level) => set((state) => {
        const own = newerLevel(state.own, level)
        return own === state.own ? state : {own}
    }),
    gained: ({gains, level}) => set((state) => {
        const pops = poppedIn(state.pops, gains)
        return {own: gainedOn(state.own, level), pops: pops === state.pops ? pops : timed(pops)}
    }),
    popOut: () => set((state) => ({pops: timed(poppedOut(state.pops))})),
    forget: () => set({own: undefined, unlocks: NO_UNLOCKS, missionRates: NO_RATES, xpRates: NO_XP_RATES, pops: timed(NO_POPS)}),
}))

export type LevelCaller = {
    accountId: string | undefined
    linked: boolean
    listenForClicks: ListenForClicks
}

// One GetLevel read at load, on each new session (a guest's first click makes one) or account, and, signed in, once its
// clicks settle. The stream's level_reached and xp_gained go to the same store, from PlayScreen.
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
