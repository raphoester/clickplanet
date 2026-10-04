import {useState} from "react"
import {Season} from "../../backends/season.ts"
import {SeasonClock, seasonClock} from "../../domain/seasonClock.ts"
import {ChevronIcon, ClockIcon} from "../components/icons.tsx"
import {useBottomEdge} from "../components/useBottomEdge.ts"
import {useEscape} from "../components/useDialog.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import SeasonFacts from "./SeasonFacts.tsx"
import {useNow} from "./useNow.ts"
import "./Season.css"

export type SeasonChipProps = {
    season: Season
    compact: boolean
    open: boolean
    onToggle: () => void
}

export default function SeasonChip({season, compact, open, onToggle}: SeasonChipProps) {
    const [chip, setChip] = useState<HTMLElement | null>(null)
    useBottomEdge(compact ? null : chip, STATUS_BOTTOM)

    const clock = seasonClock(season, useNow())
    if (!clock) return null

    const label = clock.finale ? "Final Battle ends in" : `Season ${season.number} ends in`
    const className = ["season-chip", compact ? "season-chip--compact" : "panel", clock.finale && "season-chip--live", !compact && open && "season-chip--open"]
        .filter(Boolean).join(" ")

    if (compact) {
        return <button type="button"
                       className={`panel-box ${className}`}
                       aria-label={`${label} ${clock.left}`}
                       aria-expanded={open}
                       onClick={onToggle}>
            <ClockIcon/>
            <span className="season-chip-left" aria-hidden="true">{shortLeft(clock)}</span>
        </button>
    }

    return <section ref={setChip} className={className} aria-label={clock.finale ? "Final Battle" : `Season ${season.number}`}>
        <button type="button"
                className="season-chip-head"
                aria-expanded={open}
                onClick={onToggle}>
            <ClockIcon/>
            <span className="season-chip-label">{label}</span>
            <span className="season-chip-left" role="timer">{clock.left}</span>
            <span className="season-chip-chevron"><ChevronIcon size={16}/></span>
        </button>
        {open && <SeasonPopover season={season} finale={clock.finale} onClose={onToggle}/>}
    </section>
}

function SeasonPopover({season, finale, onClose}: {season: Season, finale: boolean, onClose: () => void}) {
    useEscape(onClose)
    return <div className="season-chip-body"><SeasonFacts season={season} finale={finale}/></div>
}

export function SeasonDetails({season}: {season: Season}) {
    const clock = seasonClock(season, useNow())
    if (!clock) return null

    return <>
        <div className="panel-box season-details-left">
            <span className="menu-label">{clock.finale ? "Final Battle ends in" : "Ends in"}</span>
            <span className="season-chip-left" role="timer">{clock.left}</span>
        </div>
        <div className="panel-box season-details-facts"><SeasonFacts season={season} finale={clock.finale}/></div>
    </>
}

function shortLeft(clock: SeasonClock): string {
    return clock.left.split(" ").slice(0, 2).join(" ")
}
