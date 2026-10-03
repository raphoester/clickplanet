import {Season} from "../../backends/season.ts"
import {seasonClock} from "../../domain/seasonClock.ts"
import {useNow} from "./useNow.ts"
import "./Season.css"

export default function SeasonClock({season}: {season: Season}) {
    const clock = seasonClock(season, useNow())
    if (!clock) return null

    return <p role="timer" className={clock.finale ? "season-clock season-clock-finale" : "season-clock"}>
        {clock.line}
    </p>
}
