import {useEffect, useRef} from "react"
import {GrantRulesRefresher} from "../../backends/backend.ts"
import {useSchedule} from "../battle/scheduleStore.ts"

// The bonuses' half of the gates, their rules, read again on each turn that moves to another entry. Progression's half,
// the level and what each kind opens at, is the level's read (`useLevelFeed`).
export function useGatesFeed(grantRules: GrantRulesRefresher): void {
    const entryId = useSchedule((schedule) => schedule.turn?.entryId)
    const turned = useRef(entryId)

    useEffect(() => {
        const before = turned.current
        turned.current = entryId
        if (before === undefined || entryId === before) return
        void grantRules.refreshGrantRules().catch((e) => console.error("Could not read the bonuses' rules", e))
    }, [grantRules, entryId])
}
