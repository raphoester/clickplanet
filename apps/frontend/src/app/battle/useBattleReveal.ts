import {useCallback, useEffect, useState} from "react"
import {ClosedRound} from "../../backends/standings.ts"
import {Battle, BATTLE_SEEN_STORAGE_KEY, battleDue, battleKeyOf} from "../../domain/battle.ts"
import {DAY_BRIEF_WAIT_MS, dayBriefAwaited} from "../../domain/battleBrief.ts"

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

export function useBattleReveal(battle: Battle | undefined, closed: ClosedRound | undefined): BattleRevealState {
    const [seen, setSeen] = useState(readSeen)
    const [waited, setWaited] = useState<string>()
    const key = battle && battleKeyOf(battle)

    useEffect(() => {
        if (!key) return
        const timer = setTimeout(() => setWaited(key), DAY_BRIEF_WAIT_MS)
        return () => clearTimeout(timer)
    }, [key])

    const dismiss = useCallback(() => {
        if (!key) return
        writeSeen(key)
        setSeen(key)
    }, [key])

    const held = battle !== undefined && waited !== key && dayBriefAwaited(battle, closed)
    return {battle: battleDue(battle, seen) && !held ? battle : undefined, dismiss}
}
