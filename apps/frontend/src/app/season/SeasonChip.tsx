import {useId, useState} from "react"
import {Season} from "../../backends/season.ts"
import {countdownsOf, SeasonCountdown, seasonClock} from "../../domain/seasonClock.ts"
import {ChevronIcon, ClockIcon} from "../components/icons.tsx"
import {useBottomEdge} from "../components/useBottomEdge.ts"
import {useEscape} from "../components/useDialog.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import SeasonFacts from "./SeasonFacts.tsx"
import {useNow} from "./useNow.ts"
import {useRotation} from "./useRotation.ts"
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

    const now = useNow()
    const countdowns = countdownsOf(season, now)
    const {shown, holds} = useRotation(countdowns.length)

    const clock = seasonClock(season, now)
    if (!clock) return null

    const name = countdowns.map((countdown) => `${countdown.label} ${countdown.left}`).join(", ")
    const className = ["season-chip", compact ? "season-chip--compact" : "panel", clock.finale && "season-chip--live"]
        .filter(Boolean).join(" ")

    if (compact) {
        return <button type="button"
                       className={`panel-box ${className}`}
                       aria-label={name}
                       aria-expanded={open}
                       onClick={onToggle}
                       {...holds}>
            <ClockIcon/>
            <Faces countdowns={countdowns} shown={shown} compact/>
        </button>
    }

    return <section ref={setChip}
                    className={className}
                    aria-label={clock.finale ? "Final Battle" : `Season ${season.number}`}
                    {...holds}>
        <button type="button"
                className="season-chip-head"
                aria-label={name}
                aria-expanded={open}
                onClick={onToggle}>
            <ClockIcon/>
            <Faces countdowns={countdowns} shown={shown} compact={false}/>
            <span className="season-chip-chevron"><ChevronIcon size={16}/></span>
        </button>
        {open && <SeasonPopover season={season} finale={clock.finale} onClose={onToggle}/>}
    </section>
}

function Faces({countdowns, shown, compact}: {countdowns: SeasonCountdown[], shown: number, compact: boolean}) {
    const tagged = compact && countdowns.length > 1

    return <span className="season-chip-faces" aria-hidden="true">
        {countdowns.map((countdown, i) => <span key={countdown.label}
                                                className={i === shown ? "season-chip-face" : "season-chip-face season-chip-face--away"}>
            {compact
                ? tagged && <span className="season-chip-tag">{countdown.tag}</span>
                : <span className="season-chip-label">{countdown.label}</span>}
            <span className="season-chip-left">{compact ? shortLeft(countdown.left) : countdown.left}</span>
        </span>)}
    </span>
}

function SeasonPopover({season, finale, onClose}: {season: Season, finale: boolean, onClose: () => void}) {
    useEscape(onClose)
    return <div className="season-chip-body"><SeasonFacts season={season} finale={finale}/></div>
}

export function SeasonDetails({season}: {season: Season}) {
    const now = useNow()
    const clock = seasonClock(season, now)
    if (!clock) return null

    return <>
        <div className="panel-box season-details-countdowns">
            {countdownsOf(season, now).map((countdown) => <DetailsCountdown key={countdown.label} countdown={countdown}/>)}
        </div>
        <div className="panel-box season-details-facts"><SeasonFacts season={season} finale={clock.finale}/></div>
    </>
}

function DetailsCountdown({countdown}: {countdown: SeasonCountdown}) {
    const label = useId()

    return <div className="season-details-countdown">
        <span id={label} className="menu-label">{countdown.label}</span>
        <span className="season-chip-left" role="timer" aria-labelledby={label}>{countdown.left}</span>
    </div>
}

function shortLeft(left: string): string {
    return left.split(" ").slice(0, 2).join(" ")
}
