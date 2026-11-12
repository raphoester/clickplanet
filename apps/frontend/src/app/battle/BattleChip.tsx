import {useId, useState} from "react"
import {Season} from "../../backends/season.ts"
import {Battle, battleLeft} from "../../domain/battle.ts"
import {playingOf} from "../../domain/battleTeaser.ts"
import {countdownsOf} from "../../domain/seasonClock.ts"
import {t, translator} from "../../i18n/page.ts"
import {ChevronIcon, SwordsIcon} from "../components/icons.tsx"
import {useEdge} from "../components/useEdge.ts"
import {useEscape} from "../components/useDialog.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import {useScreen} from "../hud/screenStore.ts"
import {COUNTDOWN_WORDS} from "../season/countdownWords.ts"
import {useNow} from "../season/useNow.ts"
import {battleEndsLine, battleName} from "./battleLines.ts"
import {BattleRulesButton} from "./NextBattle.tsx"
import {useSchedule} from "./scheduleStore.ts"
import "../season/Season.css"
import "./Battle.css"

function labelOf(battle: Battle): string {
    return t(battle.endsSeason ? "finale.endsIn" : "battle.endsIn")
}

function lineOf(battle: Battle, left: string): string {
    return t(battle.endsSeason ? "finale.endsInTime" : "battle.endsInTime", {time: left})
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
    const name = lineOf(battle, translator().countdown(left))

    if (compact) {
        return <button type="button"
                       className="panel-box season-chip season-chip--compact battle-chip"
                       aria-label={name}
                       aria-expanded={open}
                       onClick={toggle}>
            <SwordsIcon/>
            <span className="season-chip-faces" aria-hidden="true">
                <span className="season-chip-face">
                    <span className="season-chip-tag">{t(battle.endsSeason ? "finale.tag" : "battle.tag")}</span>
                    <span className="season-chip-left">{translator().countdown(left, 2)}</span>
                </span>
            </span>
        </button>
    }

    const shown = open || hovered

    return <section ref={setChip}
                    className="panel season-chip battle-chip"
                    aria-label={battleName(battle)}
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
                    <span className="season-chip-left">{translator().countdown(left)}</span>
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
        {countdowns.map((countdown) => <p key={countdown.kind} className="battle-facts-season">
            <span className="menu-label">{t(COUNTDOWN_WORDS[countdown.kind].label)}</span>
            <span className="battle-facts-left">{translator().countdown(countdown.countdown)}</span>
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
                <span className="season-chip-left battle-chip-left" role="timer" aria-labelledby={label}>{translator().countdown(battleLeft(battle, now))}</span>
            </div>
        </div>
        <div className="panel-box season-details-facts"><BattleFacts battle={battle} season={season} now={now}/></div>
    </>
}
