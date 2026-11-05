import {useMemo} from "react"
import {MissionTarget, openTargetsOf} from "../../domain/missionTargets.ts"
import {MissionsState} from "./useMissions.ts"

const NONE: readonly MissionTarget[] = []

export function useMissionTargets(state: MissionsState, off: boolean): readonly MissionTarget[] {
    const key = off || state.kind !== "ready" ? "" : JSON.stringify(openTargetsOf(state.day))
    return useMemo(() => key === "" ? NONE : JSON.parse(key) as MissionTarget[], [key])
}
