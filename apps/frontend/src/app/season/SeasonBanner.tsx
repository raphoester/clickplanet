import {useState} from "react"
import {Season} from "../../backends/season.ts"
import {finaleWindow, seasonClock} from "../../domain/seasonClock.ts"
import {opensFolded} from "../compact.ts"
import {ChevronIcon} from "../components/icons.tsx"
import {useBottomEdge} from "../components/useBottomEdge.ts"
import AddToCalendarButton from "./AddToCalendarButton.tsx"
import {useNow} from "./useNow.ts"
import "./Season.css"

export const SEASON_BANNER_BOTTOM = "--season-banner-bottom"
export const SEASON_BANNER_FOLDED_KEY = "clickplanet-season-banner-folded"

export default function SeasonBanner({season}: {season: Season}) {
    const [banner, setBanner] = useState<HTMLElement | null>(null)
    const [folded, setFolded] = useState(readFolded)
    useBottomEdge(banner, SEASON_BANNER_BOTTOM)

    const clock = seasonClock(season, useNow())
    if (!clock) return null

    const toggle = () => {
        setFolded(!folded)
        writeFolded(!folded)
    }

    const head = <>
        <span className="season-banner-label">
            {clock.finale ? "Final Battle ends in" : `Season ${season.number} ends in`}
        </span>
        <span className="season-banner-left" role="timer">{clock.left}</span>
    </>

    if (clock.finale) {
        return <section ref={setBanner} className="season-banner season-banner--live" aria-label="Final Battle">
            <div className="season-banner-head">{head}</div>
        </section>
    }

    const finale = finaleWindow(season)

    return <section ref={setBanner} className="season-banner" aria-label={`Season ${season.number}`}>
        <button type="button" className="season-banner-head" aria-expanded={!folded} onClick={toggle}>
            {head}
            <span className="season-banner-chevron"><ChevronIcon size={16}/></span>
        </button>

        {!folded && <div className="season-banner-finale">
            <p className="season-banner-when">
                <span className="season-banner-finale-name">Final Battle</span>
                <span>{finale.day} · {finale.from}–{finale.to}</span>
            </p>
            <AddToCalendarButton season={season}/>
        </div>}
    </section>
}

function readFolded(): boolean {
    try {
        const stored = window.localStorage.getItem(SEASON_BANNER_FOLDED_KEY)
        return stored === null ? opensFolded() : stored === "1"
    } catch {
        return opensFolded()
    }
}

function writeFolded(folded: boolean): void {
    try {
        window.localStorage.setItem(SEASON_BANNER_FOLDED_KEY, folded ? "1" : "0")
    } catch {
        // storage unavailable
    }
}
