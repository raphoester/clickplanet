import {useId, useState} from "react"
import {Season} from "../../backends/season.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {Champion, championNames, winsLine} from "../../domain/champion.ts"
import {countdownsOf, SeasonCountdown, shortLeft} from "../../domain/seasonClock.ts"
import CountryFlag from "../components/CountryFlag.tsx"
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
    nextBattle?: BattleToCome
    compact: boolean
    open: boolean
    onToggle: () => void
}

export default function SeasonChip({season, nextBattle, compact, open, onToggle}: SeasonChipProps) {
    const [chip, setChip] = useState<HTMLElement | null>(null)
    useBottomEdge(compact ? null : chip, STATUS_BOTTOM)

    const now = useNow()
    const countdowns = countdownsOf(season, now, nextBattle)
    const {shown, holds} = useRotation(countdowns.length)

    if (countdowns.length === 0) return null

    const name = countdowns.map((countdown) => `${countdown.label} ${countdown.left}`).join(", ")
    const className = `season-chip ${compact ? "season-chip--compact" : "panel"}`

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
                    aria-label={`Season ${season.number}`}
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
        {open && <SeasonPopover onClose={onToggle}/>}
    </section>
}

export function ChampionChip({champion, compact}: {champion: Champion, compact: boolean}) {
    const [chip, setChip] = useState<HTMLElement | null>(null)
    useBottomEdge(compact ? null : chip, STATUS_BOTTOM)

    const flags = champion.countryCodes.map((code) => <CountryFlag key={code} code={code}/>)

    if (compact) {
        return <section className="panel-box season-chip season-chip--compact season-chip--champion" aria-label={winsLine(champion)}>
            <span className="season-chip-trophy" aria-hidden="true">🏆</span>
            <span className="season-chip-left" aria-hidden="true">{flags}</span>
        </section>
    }

    return <section ref={setChip} className="panel season-chip season-chip--champion" aria-label={winsLine(champion)}>
        <div className="season-chip-head" aria-hidden="true">
            <span className="season-chip-trophy">🏆</span>
            <span className="season-chip-label">Season {champion.season} champion</span>
            <span className="season-chip-left">{flags}<span className="season-chip-name">{championNames(champion)}</span></span>
        </div>
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

function SeasonPopover({onClose}: {onClose: () => void}) {
    useEscape(onClose)
    return <div className="season-chip-body"><SeasonFacts/></div>
}

export function SeasonDetails({season, nextBattle}: {season: Season, nextBattle?: BattleToCome}) {
    const countdowns = countdownsOf(season, useNow(), nextBattle)
    if (countdowns.length === 0) return null

    return <>
        <div className="panel-box season-details-countdowns">
            {countdowns.map((countdown) => <DetailsCountdown key={countdown.label} countdown={countdown}/>)}
        </div>
        <div className="panel-box season-details-facts"><SeasonFacts/></div>
    </>
}

function DetailsCountdown({countdown}: {countdown: SeasonCountdown}) {
    const label = useId()

    return <div className="season-details-countdown">
        <span id={label} className="menu-label">{countdown.label}</span>
        <span className="season-chip-left" role="timer" aria-labelledby={label}>{countdown.left}</span>
    </div>
}
