import {Season} from "../../backends/season.ts"
import {Countdown, endClock, endsLine, twoDigits} from "../../domain/seasonClock.ts"
import {Shown} from "./battle.ts"

const TICK_MS = 1000
const UNITS: (keyof Countdown)[] = ["days", "hours", "minutes", "seconds"]

// The season that runs and the time left in it, following its moves, until it ends.
export function seasonShown(root: ParentNode): Shown<Season | undefined> {
    const shown = [...root.querySelectorAll<HTMLElement>("[data-season]")]
    let timer: ReturnType<typeof setInterval> | undefined

    const stop = () => clearInterval(timer)
    const show = (season: Season | undefined) => {
        stop()
        const tick = (): boolean => {
            const clock = season && endClock(season, Date.now())
            for (const element of shown) element.hidden = !clock
            if (!season || !clock) return false

            write(root, "season-left", clock.left)
            for (const unit of UNITS) write(root, `countdown-${unit}`, twoDigits(clock.countdown[unit]))
            return true
        }

        if (season) {
            write(root, "season-number", String(season.number))
            write(root, "season-ends", endsLine(season))
        }
        if (!tick()) return
        timer = setInterval(() => {
            if (!tick()) stop()
        }, TICK_MS)
    }
    return {show, stop}
}

function write(root: ParentNode, field: string, text: string) {
    for (const element of root.querySelectorAll(`[data-${field}]`)) element.textContent = text
}
