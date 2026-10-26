import {useCallback, useState} from "react"
import {Battle, BATTLE_SEEN_STORAGE_KEY, battleDue, battleKeyOf} from "../../domain/battle.ts"

function readSeen(): string | null {
    try {
        return window.localStorage.getItem(BATTLE_SEEN_STORAGE_KEY)
    } catch {
        return null
    }
}

function writeSeen(key: string) {
    try {
        window.localStorage.setItem(BATTLE_SEEN_STORAGE_KEY, key)
    } catch (e) {
        console.error("Could not remember the battle shown", e)
    }
}

export type BattleRevealState = {
    battle?: Battle
    dismiss: () => void
}

export function useBattleReveal(battle: Battle | undefined): BattleRevealState {
    const [seen, setSeen] = useState(readSeen)

    const dismiss = useCallback(() => {
        if (!battle) return
        const key = battleKeyOf(battle)
        writeSeen(key)
        setSeen(key)
    }, [battle])

    return {battle: battleDue(battle, seen) ? battle : undefined, dismiss}
}
