import {create} from "zustand"
import {NO_SEASONS, Season, Seasons} from "../../backends/season.ts"
import {ClosedRound} from "../../backends/standings.ts"
import {BattleToCome, Debrief, Turn} from "../../backends/turn.ts"
import {Battle, battleOf} from "../../domain/battle.ts"

type Schedule = {
    turn?: Turn
    battle?: Battle
    battles: BattleToCome[]
    seasons: Seasons
    debrief?: Debrief
    // The last debrief's numbers, once read from scoring.
    closed?: ClosedRound
    // The current season, once its end passed on this page's clock.
    ended?: Season
}

type ScheduleStore = Schedule & {
    hearTurn: (turn: Turn) => void
    hearBattles: (battles: BattleToCome[]) => void
    hearSeasons: (seasons: Seasons) => void
    hearDebrief: (debrief: Debrief | undefined) => void
    readDebrief: (debrief: Debrief, closed: ClosedRound) => void
    endSeason: (season: Season) => void
}

export const NO_SCHEDULE: Schedule = {
    turn: undefined,
    battle: undefined,
    battles: [],
    seasons: NO_SEASONS,
    debrief: undefined,
    closed: undefined,
    ended: undefined,
}

export const useSchedule = create<ScheduleStore>()((set) => ({
    ...NO_SCHEDULE,
    hearTurn: (turn) => set({turn, battle: battleOf(turn)}),
    hearBattles: (battles) => set({battles}),
    hearSeasons: (seasons) => set({seasons}),
    // A reconnect sends the same debrief again: its numbers are not read twice.
    hearDebrief: (debrief) => set((schedule) => sameDebrief(schedule.debrief, debrief) ? schedule : {debrief, closed: undefined}),
    readDebrief: (debrief, closed) => set((schedule) => schedule.debrief === debrief ? {closed} : schedule),
    endSeason: (season) => set({ended: season}),
}))

export function runningSeason(schedule: Schedule): Season | undefined {
    return schedule.seasons.current === schedule.ended ? undefined : schedule.seasons.current
}

function sameDebrief(a: Debrief | undefined, b: Debrief | undefined): boolean {
    return JSON.stringify(a) === JSON.stringify(b)
}
