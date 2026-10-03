import {useEffect, useState} from "react"
import {Season} from "../../backends/season.ts"
import {seasonClock} from "../../domain/seasonClock.ts"
import "./Season.css"

const TICK_MS = 1000

export default function SeasonClock({season}: {season: Season}) {
    const [now, setNow] = useState(Date.now)

    useEffect(() => {
        const timer = setInterval(() => setNow(Date.now()), TICK_MS)
        return () => clearInterval(timer)
    }, [])

    const clock = seasonClock(season, now)
    if (!clock) return null

    return <p role="timer" className={clock.finale ? "season-clock season-clock-finale" : "season-clock"}>
        {clock.text}
    </p>
}
