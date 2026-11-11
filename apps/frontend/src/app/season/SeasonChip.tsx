import {useId, useState} from "react"
import {Season} from "../../backends/season.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {Finale, finaleNear} from "../../domain/battle.ts"
import {Champion, championNames, winsLine} from "../../domain/champion.ts"
import {countdownsOf, SeasonCountdown, shortLeft} from "../../domain/seasonClock.ts"
import {BattleRulesButton, NextBattleDetails} from "../battle/NextBattle.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {ChevronIcon, ClockIcon} from "../components/icons.tsx"
import {useEdge} from "../components/useEdge.ts"
import {useEscape} from "../components/useDialog.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import {useScreen} from "../hud/screenStore.ts"
import {FinaleDetails} from "./FinaleFacts.tsx"
import SeasonFacts from "./SeasonFacts.tsx"
import {useNow} from "./useNow.ts"
import {useRotation} from "./useRotation.ts"
import "./Season.css"

export type SeasonChipProps = {
    season: Season
    nextBattle?: BattleToCome
    finale?: Finale
    dayEndsAt?: number
    compact: boolean
}

export default function SeasonChip({season, nextBattle, finale, dayEndsAt, compact}: SeasonChipProps) {
    const [chip, setChip] = useState<HTMLElement | null>(null)
    useEdge(compact ? null : chip, STATUS_BOTTOM, "bottom")
    const open = useScreen((screen) => compact ? screen.sheet === "season" : screen.popovers.season)
    const toggleSheet = useScreen((screen) => screen.toggleSheet)
    const togglePopover = useScreen((screen) => screen.togglePopover)
    const toggle = () => compact ? toggleSheet("season") : togglePopover("season")

    const now = useNow()
    const countdowns = countdownsOf(season, now, nextBattle, dayEndsAt)
    const {shown, holds} = useRotation(countdowns.length)

    if (countdowns.length === 0) return null

    const name = countdowns.map((countdown) => `${countdown.label} ${countdown.left}`).join(", ")
    const near = finaleNear(finale, now) ? " season-chip--final" : ""
    const className = `season-chip ${compact ? "season-chip--compact" : "panel"}${near}`

    if (compact) {
        return <button type="button"
                       className={`panel-box ${className}`}
                       aria-label={name}
                       aria-expanded={open}
                       onClick={toggle}
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
                onClick={toggle}>
            <ClockIcon/>
            <Faces countdowns={countdowns} shown={shown} compact={false}/>
            <span className="season-chip-chevron"><ChevronIcon size={16}/></span>
        </button>
        {open && <SeasonPopover nextBattle={nextBattle} finale={finale} onClose={toggle}/>}
    </section>
}

export function ChampionChip({champion, compact}: {champion: Champion, compact: boolean}) {
    const [chip, setChip] = useState<HTMLElement | null>(null)
    useEdge(compact ? null : chip, STATUS_BOTTOM, "bottom")

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
                                                className={faceClassOf(countdown, i === shown)}>
            {compact
                ? tagged && <span className="season-chip-tag">{countdown.tag}</span>
                : <span className="season-chip-label">{countdown.label}</span>}
            <span className="season-chip-left">{compact ? shortLeft(countdown.left) : countdown.left}</span>
        </span>)}
    </span>
}

function faceClassOf(countdown: SeasonCountdown, shown: boolean): string {
    const final = countdown.endsSeason ? " season-chip-face--final" : ""
    return `season-chip-face${final}${shown ? "" : " season-chip-face--away"}`
}

function SeasonPopover({nextBattle, finale, onClose}: {nextBattle?: BattleToCome, finale?: Finale, onClose: () => void}) {
    useEscape(onClose)
    return <div className="season-chip-body">
        {nextBattle && !nextBattle.endsSeason && <NextBattleDetails battle={nextBattle}/>}
        {finale && !finale.live && <FinaleDetails finale={finale}><FinaleRules nextBattle={nextBattle}/></FinaleDetails>}
        <SeasonFacts/>
    </div>
}

// The Final Battle's rules, once it is the next battle: its teaser's cards are its rule set's, as any battle's.
function FinaleRules({nextBattle}: {nextBattle?: BattleToCome}) {
    return nextBattle?.endsSeason ? <BattleRulesButton battle={nextBattle}/> : null
}

export type SeasonDetailsProps = {
    season: Season
    nextBattle?: BattleToCome
    finale?: Finale
    dayEndsAt?: number
}

export function SeasonDetails({season, nextBattle, finale, dayEndsAt}: SeasonDetailsProps) {
    const countdowns = countdownsOf(season, useNow(), nextBattle, dayEndsAt)
    if (countdowns.length === 0) return null

    return <>
        <div className="panel-box season-details-countdowns">
            {countdowns.map((countdown) => <DetailsCountdown key={countdown.label} countdown={countdown}/>)}
        </div>
        {nextBattle && !nextBattle.endsSeason && <div className="panel-box season-details-facts">
            <NextBattleDetails battle={nextBattle}/>
        </div>}
        {finale && !finale.live && <div className="panel-box season-details-facts">
            <FinaleDetails finale={finale}><FinaleRules nextBattle={nextBattle}/></FinaleDetails>
        </div>}
        <div className="panel-box season-details-facts"><SeasonFacts/></div>
    </>
}

function DetailsCountdown({countdown}: {countdown: SeasonCountdown}) {
    const label = useId()

    return <div className={countdown.endsSeason ? "season-details-countdown season-chip-face--final" : "season-details-countdown"}>
        <span id={label} className="menu-label">{countdown.label}</span>
        <span className="season-chip-left" role="timer" aria-labelledby={label}>{countdown.left}</span>
    </div>
}
