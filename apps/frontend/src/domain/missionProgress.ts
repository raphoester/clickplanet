import {DayMissions, FlagMissionOrder, MissionOrder} from "../backends/missions.ts"
import {reachedOf} from "./missions.ts"

export type Moved = {
    id: string
    order: MissionOrder | FlagMissionOrder
    flag: boolean
}

// The bank for the flag moves on every take: on the line it would never go.
export function movedOf(before: DayMissions, after: DayMissions): Moved[] {
    const moved: Moved[] = []
    for (const mission of after.missions) {
        const was = before.missions.find(({id}) => id === mission.id)
        if (!was || mission.done || mission.order.kind === "takeFor") continue
        if (reachedOf(mission.order) > reachedOf(was.order)) moved.push({id: mission.id, order: mission.order, flag: false})
    }

    const flag = after.flagMission
    const flagWas = before.flagMission
    if (flag && flagWas && flag.id === flagWas.id && !flag.done && reachedOf(flag.order) > reachedOf(flagWas.order)) {
        moved.push({id: flag.id, order: flag.order, flag: true})
    }
    return moved
}

export function nearestDone(moved: readonly Moved[]): Moved | undefined {
    let nearest: Moved | undefined
    for (const one of moved) {
        if (!nearest || shareOf(one) > shareOf(nearest)) nearest = one
    }
    return nearest
}

function shareOf({order}: Moved): number {
    return reachedOf(order) / order.goal
}

export function stillOpen(id: string, day: DayMissions): boolean {
    if (day.flagMission?.id === id) return !day.flagMission.done
    const mission = day.missions.find((one) => one.id === id)
    return mission !== undefined && !mission.done
}
