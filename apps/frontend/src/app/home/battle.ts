import {BattleToCome} from "../../backends/turn.ts"
import {nextBattleOf} from "../../domain/battle.ts"
import {timeLeftTo} from "../../domain/seasonClock.ts"

const TICK_MS = 1000

export type Shown<T> = {
    show: (value: T) => void
    stop: () => void
}

// The time left before the next battle, following the schedule's edits; hidden while none is to come.
export function nextBattleShown(root: ParentNode): Shown<BattleToCome[]> {
    const shown = [...root.querySelectorAll<HTMLElement>("[data-battle]")]
    let battles: BattleToCome[] = []

    const tick = () => {
        const now = Date.now()
        const next = nextBattleOf(battles, now)
        for (const element of shown) element.hidden = !next
        if (!next) return
        for (const element of root.querySelectorAll("[data-battle-left]")) element.textContent = timeLeftTo(next.startsAt, now)
    }

    const timer = setInterval(tick, TICK_MS)
    return {
        show: (next) => {
            battles = next
            tick()
        },
        stop: () => clearInterval(timer),
    }
}
