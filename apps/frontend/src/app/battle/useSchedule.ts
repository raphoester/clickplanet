import {useEffect, useState} from "react"
import {Seasons} from "../../backends/season.ts"
import {BattleToCome, Turn, TurnBackend} from "../../backends/turn.ts"

export type Schedule = {
    turn?: Turn
    battles: BattleToCome[]
    seasons: Seasons
}

export function useSchedule(backend: TurnBackend | undefined): Schedule {
    const [turn, setTurn] = useState<Turn>()
    const [battles, setBattles] = useState<BattleToCome[]>([])
    const [seasons, setSeasons] = useState<Seasons>({})

    useEffect(() => backend?.follow({onTurn: setTurn, onBattles: setBattles, onSeasons: setSeasons}), [backend])

    return {turn, battles, seasons}
}
