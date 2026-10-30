import {useEffect, useRef} from "react"
import {DayMissions, FlagMission} from "../../backends/missions.ts"
import {newlyDoneFlagMission} from "../../domain/missions.ts"
import {MissionsState} from "./useMissions.ts"

// The first read says nothing: a mission done before the page opened was cheered then.
export function useFlagMissionDone(state: MissionsState, onDone: (mission: FlagMission) => void): void {
    const last = useRef<DayMissions | undefined>(undefined)
    const latest = useRef(onDone)
    latest.current = onDone

    useEffect(() => {
        if (state.kind !== "ready") return
        const before = last.current
        last.current = state.day
        if (!before) return
        const done = newlyDoneFlagMission(before, state.day)
        if (done) latest.current(done)
    }, [state])
}
