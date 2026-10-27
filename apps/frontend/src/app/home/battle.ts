import {BattleToCome, TurnBackend} from "../../backends/turn.ts"
import {nextBattleOf} from "../../domain/battle.ts"
import {timeLeftTo} from "../../domain/seasonClock.ts"

const TICK_MS = 1000

// The time left before the next battle, following the schedule's edits; hidden while none is to come.
export function showNextBattle(root: ParentNode, turns: TurnBackend): () => void {
    const shown = [...root.querySelectorAll<HTMLElement>("[data-battle]")]
    let battles: BattleToCome[] = []

    const tick = () => {
        const now = Date.now()
        const next = nextBattleOf(battles, now)
        for (const element of shown) element.hidden = !next
        if (!next) return
        for (const element of root.querySelectorAll("[data-battle-left]")) element.textContent = timeLeftTo(next.startsAt, now)
    }

    const stop = turns.follow({
        onTurn: () => {},
        onBattles: (next) => {
            battles = next
            tick()
        },
    })
    const timer = setInterval(tick, TICK_MS)
    return () => {
        stop()
        clearInterval(timer)
    }
}
