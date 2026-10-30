import {BattleToCome, PLAIN_TURN, Turn} from "../../backends/turn.ts"
import {finaleLengthLine, finaleOf, finaleWhenLine} from "../../domain/battle.ts"
import {Countdown, countdownTo, twoDigits} from "../../domain/seasonClock.ts"
import {write} from "./season.ts"

const TICK_MS = 1000
const UNITS: (keyof Countdown)[] = ["days", "hours", "minutes", "seconds"]

export type FinaleShown = {
    showBattles: (battles: BattleToCome[]) => void
    showTurn: (turn: Turn) => void
    stop: () => void
}

// The season's Final Battle: when it plays, what it is worth, and the time to its start, then to its end.
export function finaleShown(root: ParentNode): FinaleShown {
    const panels = [...root.querySelectorAll<HTMLElement>("[data-finale]")]
    let battles: BattleToCome[] = []
    let turn: Turn = PLAIN_TURN

    const tick = () => {
        const now = Date.now()
        const finale = finaleOf(battles, turn, now)
        for (const panel of panels) panel.hidden = !finale
        if (!finale) return

        for (const panel of panels) panel.toggleAttribute("data-live", finale.live)
        const countdown = countdownTo(finale.live ? finale.endsAt : finale.startsAt, now)
        for (const unit of UNITS) write(root, `finale-${unit}`, twoDigits(countdown[unit]))
        write(root, "finale-when", finaleWhenLine(finale))
        write(root, "finale-length", finaleLengthLine(finale))
        write(root, "finale-times", String(finale.times))
        write(root, "finale-words", finale.description)
        showIf(root, "finale-times-row", finale.times > 1)
        showIf(root, "finale-rules", finale.description !== "")
    }

    const timer = setInterval(tick, TICK_MS)
    return {
        showBattles: (next) => {
            battles = next
            tick()
        },
        showTurn: (next) => {
            turn = next
            tick()
        },
        stop: () => clearInterval(timer),
    }
}

function showIf(root: ParentNode, field: string, shown: boolean) {
    for (const element of root.querySelectorAll<HTMLElement>(`[data-${field}]`)) element.hidden = !shown
}
