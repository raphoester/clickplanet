import {useEffect, useState} from "react"
import {NO_SEASONS, Seasons} from "../../backends/season.ts"
import {BattleToCome, Debrief, Turn, TurnBackend} from "../../backends/turn.ts"

export type Schedule = {
    turn?: Turn
    battles: BattleToCome[]
    seasons: Seasons
    debrief?: Debrief
}

export function useSchedule(backend: TurnBackend | undefined): Schedule {
    const [turn, setTurn] = useState<Turn>()
    const [battles, setBattles] = useState<BattleToCome[]>([])
    const [seasons, setSeasons] = useState<Seasons>(NO_SEASONS)
    const [debrief, setDebrief] = useState<Debrief>()

    useEffect(() => backend?.follow({
        onTurn: setTurn,
        onBattles: setBattles,
        onSeasons: setSeasons,
        onDebrief: (next) => setDebrief((was) => sameDebrief(was, next) ? was : next),
    }), [backend])

    return {turn, battles, seasons, debrief}
}

// A reconnect sends the same debrief again: its numbers are not read twice.
function sameDebrief(a: Debrief | undefined, b: Debrief | undefined): boolean {
    return JSON.stringify(a) === JSON.stringify(b)
}
