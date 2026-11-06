import {useEffect, useRef, useState} from "react"
import {DayMissions} from "../../backends/missions.ts"
import {Moved, movedOf, nearestDone, stillOpen} from "../../domain/missionProgress.ts"
import {MissionsState} from "./missionsStore.ts"

export const PROGRESS_LINE_MS = 3000

export function useMissionProgress(state: MissionsState, off: boolean): Moved | undefined {
    const [line, setLine] = useState<Moved>()
    const last = useRef<DayMissions | undefined>(undefined)

    useEffect(() => {
        if (state.kind !== "ready") return
        const before = last.current
        last.current = state.day
        if (!before || off) return
        const moved = movedOf(before, state.day)
        setLine((shown) => {
            if (!shown) return nearestDone(moved)
            if (!stillOpen(shown.id, state.day)) return undefined
            return moved.find(({id}) => id === shown.id) ?? shown
        })
    }, [state, off])

    useEffect(() => {
        if (!line) return
        const timer = setTimeout(() => setLine(undefined), PROGRESS_LINE_MS)
        return () => clearTimeout(timer)
    }, [line])

    return off ? undefined : line
}
