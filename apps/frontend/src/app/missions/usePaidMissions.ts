import {useEffect, useRef} from "react"
import {Mission} from "../../backends/missions.ts"
import {newlyPaid} from "../../domain/missions.ts"
import {MissionsState} from "./useMissions.ts"

// The first read says nothing: what was paid before the page opened was shown then.
export function usePaidMissions(state: MissionsState, onPaid: (paid: Mission[]) => void): void {
    const last = useRef<readonly Mission[] | undefined>(undefined)
    const latest = useRef(onPaid)
    latest.current = onPaid

    useEffect(() => {
        if (state.kind !== "ready") return
        const before = last.current
        last.current = state.day.missions
        if (!before) return
        const paid = newlyPaid(before, state.day.missions)
        if (paid.length > 0) latest.current(paid)
    }, [state])
}
