import {useEffect, useRef, useState} from "react"
import {create} from "zustand"
import {Ladder, NO_LADDER} from "../../domain/bonus.ts"
import {GrantRulesRefresher, SessionSource} from "../../backends/backend.ts"
import {LadderReader} from "../../backends/progression.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useReadsAfterClicks} from "../clicks/useAcceptedClicks.ts"
import {useSchedule} from "../battle/scheduleStore.ts"

export const useLadder = create<{ladder: Ladder}>()(() => ({ladder: NO_LADDER}))

export type GatesSources = {
    ladder: LadderReader
    grantRules: GrantRulesRefresher
    sessions: SessionSource
}

export type GatesCaller = {
    accountId: string | undefined
    linked: boolean
    listenForClicks: ListenForClicks
}

// The ladder at load, on each new session or account and, signed in, once its clicks settle; the bonuses' rules on each
// turn that moves to another entry.
export function useGatesFeed({ladder: reader, grantRules, sessions}: GatesSources, {accountId, linked, listenForClicks}: GatesCaller): void {
    const reads = useReadsAfterClicks(linked ? listenForClicks : undefined)
    const [session, setSession] = useState<string>()
    const entryId = useSchedule((schedule) => schedule.turn?.entryId)
    const turned = useRef(entryId)

    useEffect(() => sessions.listenForSessions(setSession), [sessions])

    useEffect(() => {
        let stale = false
        reader.ladder().then(
            (ladder) => {
                if (!stale && ladder) useLadder.setState({ladder})
            },
            (e) => console.error("Could not read the ladder", e),
        )
        return () => {
            stale = true
        }
    }, [reader, session, accountId, reads])

    useEffect(() => {
        const before = turned.current
        turned.current = entryId
        if (before === undefined || entryId === before) return
        void grantRules.refreshGrantRules().catch((e) => console.error("Could not read the bonuses' rules", e))
    }, [grantRules, entryId])

    useEffect(() => () => useLadder.setState({ladder: NO_LADDER}), [accountId])
}
