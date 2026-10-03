import {Season} from "../../backends/season.ts"
import {finaleWindow, seasonClock} from "../../domain/seasonClock.ts"
import AddToCalendarButton from "./AddToCalendarButton.tsx"
import {useNow} from "./useNow.ts"
import "./Season.css"

export default function SeasonCard({season, titleId}: {season: Season, titleId: string}) {
    const clock = seasonClock(season, useNow())
    if (!clock) return null

    const finale = finaleWindow(season)

    return <div className={clock.finale ? "season-card season-card-live" : "season-card"}>
        <div className="season-card-head">
            <h2 className="season-card-title" id={titleId}>Season {season.number}</h2>
            <p className="season-card-clock" role="timer">
                <span className="menu-label">Ends in</span>
                <span className="season-card-left">{clock.left}</span>
            </p>
        </div>

        <div className="season-card-finale">
            <p className="season-card-when">
                <span className="menu-label season-card-finale-name">Final Assault</span>
                <span>{clock.finale ? `Now · until ${finale.to}` : `${finale.day} · ${finale.from}–${finale.to}`}</span>
            </p>
            {!clock.finale && <AddToCalendarButton season={season}/>}
        </div>
    </div>
}
