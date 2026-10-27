import {useCallback, useEffect, useRef, useState} from "react"
import {Battle} from "../../domain/battle.ts"

export type BattleOverState = {
    over?: Battle
    dismiss: () => void
}

export function useBattleOver(battle: Battle | undefined): BattleOverState {
    const [over, setOver] = useState<Battle>()
    const previous = useRef(battle)

    useEffect(() => {
        const was = previous.current
        previous.current = battle
        if (was && !battle) setOver(was)
    }, [battle])

    const dismiss = useCallback(() => setOver(undefined), [])

    return {over, dismiss}
}
