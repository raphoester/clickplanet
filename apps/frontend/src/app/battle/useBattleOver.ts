import {useCallback, useEffect, useRef, useState} from "react"
import {Debrief} from "../../backends/turn.ts"
import {Battle} from "../../domain/battle.ts"
import {briefAwaited} from "../../domain/battleBrief.ts"

export type BattleOverState = {
    over?: Battle
    awaiting: boolean
    dismiss: () => void
}

export function useBattleOver(battle: Battle | undefined, debrief: Debrief | undefined): BattleOverState {
    const [over, setOver] = useState<Battle>()
    const previous = useRef(battle)

    useEffect(() => {
        const was = previous.current
        previous.current = battle
        if (was && !battle) setOver(was)
    }, [battle])

    const dismiss = useCallback(() => setOver(undefined), [])

    return {over, awaiting: over !== undefined && briefAwaited(over, debrief), dismiss}
}
