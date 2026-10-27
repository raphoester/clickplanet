import {useEffect, useState} from "react"
import {BattleToCome, Turn, TurnBackend} from "../../backends/turn.ts"

export type Schedule = {
    turn?: Turn
    battles: BattleToCome[]
}

export function useSchedule(backend: TurnBackend | undefined): Schedule {
    const [turn, setTurn] = useState<Turn>()
    const [battles, setBattles] = useState<BattleToCome[]>([])

    useEffect(() => backend?.follow({onTurn: setTurn, onBattles: setBattles}), [backend])

    return {turn, battles}
}
