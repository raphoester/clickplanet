import {Season} from "../../backends/season.ts"
import {Countdown, finaleClock, finaleWindow, twoDigits} from "../../domain/seasonClock.ts"

const TICK_MS = 1000
const LIVE = "finale-live"
const UNITS: (keyof Countdown)[] = ["days", "hours", "minutes", "seconds"]

export function showFinale(root: ParentNode, season: Season): () => void {
    const when = finaleWindow(season)
    write(root, "season-number", String(season.number))
    write(root, "finale-date", when.day)
    write(root, "finale-time", `${when.from}–${when.to}`)

    const shown = [...root.querySelectorAll<HTMLElement>("[data-finale]")]
    const tick = (): boolean => {
        const clock = finaleClock(season, Date.now())
        for (const element of shown) {
            element.hidden = !clock
            element.classList.toggle(LIVE, clock?.live ?? false)
        }
        if (!clock) return false

        write(root, "finale-left", clock.left)
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
