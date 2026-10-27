import {Season} from "../../backends/season.ts"
import {Countdown, endClock, endsLine, twoDigits} from "../../domain/seasonClock.ts"

const TICK_MS = 1000
const UNITS: (keyof Countdown)[] = ["days", "hours", "minutes", "seconds"]

// The season and the time left in it, until it ends.
export function showSeason(root: ParentNode, season: Season): () => void {
    write(root, "season-number", String(season.number))
    write(root, "season-ends", endsLine(season))

    const shown = [...root.querySelectorAll<HTMLElement>("[data-season]")]
    const tick = (): boolean => {
        const clock = endClock(season, Date.now())
        for (const element of shown) element.hidden = !clock
        if (!clock) return false

        write(root, "season-left", clock.left)
        for (const unit of UNITS) write(root, `countdown-${unit}`, twoDigits(clock.countdown[unit]))
        return true
    }

    if (!tick()) return () => {}
    const timer = setInterval(() => {
        if (!tick()) clearInterval(timer)
    }, TICK_MS)
    return () => clearInterval(timer)
}

function write(root: ParentNode, field: string, text: string) {
    for (const element of root.querySelectorAll(`[data-${field}]`)) element.textContent = text
}
