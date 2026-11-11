import {useId, useState} from "react"
import {Season} from "../../backends/season.ts"
import {Battle, battleEndsLine, battleLeft, battleNameOf} from "../../domain/battle.ts"
import {playingOf} from "../../domain/battleTeaser.ts"
import {countdownsOf, shortLeft} from "../../domain/seasonClock.ts"
import {ChevronIcon, SwordsIcon} from "../components/icons.tsx"
import {useEdge} from "../components/useEdge.ts"
import {useEscape} from "../components/useDialog.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import {useScreen} from "../hud/screenStore.ts"
import {useNow} from "../season/useNow.ts"
import {BattleRulesButton} from "./NextBattle.tsx"
import {useSchedule} from "./scheduleStore.ts"
import "../season/Season.css"
import "./Battle.css"

function labelOf(battle: Battle): string {
    return `${battleNameOf(battle)} ends in`
}

export type BattleChipProps = {
    battle: Battle
    season?: Season
    compact: boolean
}

export default function BattleChip({battle, season, compact}: BattleChipProps) {
    const [chip, setChip] = useState<HTMLElement | null>(null)
    useEdge(compact ? null : chip, STATUS_BOTTOM, "bottom")
    const [hovered, setHovered] = useState(false)
    const open = useScreen((screen) => compact ? screen.sheet === "battle" : screen.popovers.battle)
    const toggleSheet = useScreen((screen) => screen.toggleSheet)
    const togglePopover = useScreen((screen) => screen.togglePopover)
    const toggle = () => compact ? toggleSheet("battle") : togglePopover("battle")

    const now = useNow()
    const left = battleLeft(battle, now)
    const name = `${labelOf(battle)} ${left}`

    if (compact) {
        return <button type="button"
                       className="panel-box season-chip season-chip--compact battle-chip"
                       aria-label={name}
                       aria-expanded={open}
                       onClick={toggle}>
            <SwordsIcon/>
            <span className="season-chip-faces" aria-hidden="true">
                <span className="season-chip-face">
                    <span className="season-chip-tag">{battle.endsSeason ? "Final" : "Battle"}</span>
                    <span className="season-chip-left">{shortLeft(left)}</span>
                </span>
            </span>
        </button>
    }

    const shown = open || hovered

    return <section ref={setChip}
                    className="panel season-chip battle-chip"
                    aria-label={battleNameOf(battle)}
                    onPointerEnter={() => setHovered(true)}
                    onPointerLeave={() => setHovered(false)}>
        <button type="button"
                className="season-chip-head"
                aria-label={name}
                aria-expanded={shown}
                onClick={toggle}>
            <SwordsIcon/>
            <span className="season-chip-faces" aria-hidden="true">
                <span className="season-chip-face">
                    <span className="season-chip-label">{labelOf(battle)}</span>
                    <span className="season-chip-left">{left}</span>
                </span>
            </span>
            <span className="season-chip-chevron"><ChevronIcon size={16}/></span>
        </button>
        {shown && <BattlePopover battle={battle} season={season} now={now} open={open} onClose={toggle}/>}
    </section>
}

type BattlePopoverProps = {
    battle: Battle
    season?: Season
    now: number
    open: boolean
    onClose: () => void
}

function BattlePopover({battle, season, now, open, onClose}: BattlePopoverProps) {
    useEscape(() => {
        if (open) onClose()
    })
    return <div className="season-chip-body"><BattleFacts battle={battle} season={season} now={now}/></div>
}

export function BattleFacts({battle, season, now}: {battle: Battle, season?: Season, now: number}) {
    const countdowns = season ? countdownsOf(season, now) : []
    const turn = useSchedule((schedule) => schedule.turn)

    return <div className="battle-facts">
        {battle.description && <p className="battle-facts-text">{battle.description}</p>}
        <p className="battle-facts-when">{battleEndsLine(battle, now)}</p>
        <BattleRulesButton battle={playingOf(turn)}/>
        {countdowns.map((countdown) => <p key={countdown.label} className="battle-facts-season">
            <span className="menu-label">{countdown.label}</span>
            <span className="battle-facts-left">{countdown.left}</span>
        </p>)}
    </div>
}

export function BattleDetails({battle, season}: {battle: Battle, season?: Season}) {
    const now = useNow()
    const label = useId()

    return <>
        <div className="panel-box season-details-countdowns">
            <div className="season-details-countdown">
                <span id={label} className="menu-label">{labelOf(battle)}</span>
                <span className="season-chip-left battle-chip-left" role="timer" aria-labelledby={label}>{battleLeft(battle, now)}</span>
            </div>
        </div>
        <div className="panel-box season-details-facts"><BattleFacts battle={battle} season={season} now={now}/></div>
    </>
}
