import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {ClosedRound, Standing} from "../../backends/standings.ts"
import {BATTLE_DEBRIEF, DebriefScreen, PointsRow, pointsRowsOf} from "../../domain/battleBrief.ts"
import {Countries} from "../../domain/countries.ts"
import {battleFinishedLine} from "../../domain/roundReveal.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {SwordsIcon} from "../components/icons.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import TitleBadge from "../titles/TitleBadge.tsx"
import "../titles/TitleUnlocked.css"
import "../season/RoundReveal.css"
import "./Battle.css"

export type BattleDebriefProps = {
    screen: DebriefScreen
    closed: ClosedRound
    description: string
    countryCode: string
    accountId?: string
    // Another screen follows this one.
    more: boolean
    play: PlaySound
    onClose: () => void
}

export default function BattleDebrief({screen, closed, description, countryCode, accountId, more, play, onClose}: BattleDebriefProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const [ready, setReady] = useState(false)

    useEffect(() => {
        if (screen === "over") play("title")
        const timer = setTimeout(() => setReady(true), BATTLE_DEBRIEF[screen] * 1000)
        return () => clearTimeout(timer)
    }, [play, screen])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    return createPortal(
        <div ref={dialog}
             className="title-reveal round-reveal battle-brief"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}>
            <div className="round-reveal-stage battle-brief-stage">
                {screen === "over" && <Over headId={headId} closed={closed} description={description}/>}
                {screen === "points" && <Points headId={headId} closed={closed} rows={pointsRowsOf(closed, countryCode)}/>}
                {screen === "players" && <Players headId={headId} players={closed.players} accountId={accountId}/>}

                <div className="title-reveal-actions round-reveal-actions">
                    {ready && <button type="button" className={more ? "button" : "button button-gold"} onClick={onClose}>
                        {more ? "Next" : "To the globe"}
                    </button>}
                </div>
            </div>
            {screen === "over" && <div className="title-reveal-flash" aria-hidden="true"/>}
        </div>,
        document.body,
    )
}

function Over({headId, closed, description}: {headId: string, closed: ClosedRound, description: string}) {
    return <>
        <h2 id={headId} className="title-reveal-head round-reveal-head">{battleFinishedLine(closed)}</h2>
        <div className="title-reveal-medal">
            <div className="title-reveal-rays" aria-hidden="true"/>
            <div className="title-reveal-wave" aria-hidden="true"/>
            <span className="title-reveal-float">
                <span className="battle-reveal-disc" aria-hidden="true"><SwordsIcon size={110}/></span>
            </span>
        </div>
        {description && <p className="battle-brief-text">{description}</p>}
    </>
}

function Points({headId, closed, rows}: {headId: string, closed: ClosedRound, rows: PointsRow[]}) {
    return <>
        <h2 id={headId} className="title-reveal-head round-reveal-head">Points won</h2>
        {closed.times > 1 && <p className="battle-brief-times">
            <SwordsIcon size={20}/>
            <span>Points ×{closed.times}</span>
        </p>}
        <ol className="battle-brief-list" aria-label="Points won">
            {rows.map((row, i) => <li key={row.countryCode}
                                      className={row.mine ? "battle-brief-row battle-brief-row-mine" : "battle-brief-row"}
                                      style={{"--share": row.width, "--enter": i} as CSSProperties}
                                      aria-current={row.mine ? "true" : undefined}>
                <span className="round-reveal-rank">{row.rank}</span>
                <CountryFlag code={row.countryCode}/>
                <span className="battle-brief-name">{nameOf(row.countryCode)}</span>
                {closed.times > 1 && <span className="battle-brief-times-of">{row.won} ×{closed.times}</span>}
                <span className="battle-brief-tiles">{row.points}</span>
            </li>)}
        </ol>
    </>
}

function Players({headId, players, accountId}: {headId: string, players: Standing[], accountId?: string}) {
    return <>
        <h2 id={headId} className="title-reveal-head round-reveal-head">Top players</h2>
        <ol className="battle-brief-list" aria-label="Top players">
            {players.map((player, i) => {
                const you = player.accountId === accountId
                return <li key={player.accountId}
                           className={you ? "battle-brief-row battle-brief-row-mine" : "battle-brief-row"}
                           style={{"--enter": i} as CSSProperties}
                           aria-current={you ? "true" : undefined}>
                    <span className="round-reveal-rank">{player.rank}</span>
                    <span className="battle-brief-flag" role="img" aria-label={nameOf(player.countryCode)}>
                        <CountryFlag code={player.countryCode}/>
                    </span>
                    <span className="battle-brief-name battle-brief-player"
                          style={authorStyle({color: player.color, guest: false})}
                          title={player.name}>{player.name}</span>
                    <TitleBadge title={player.wornTitle} size={18}/>
                    <span className="battle-brief-tiles">{player.tiles}</span>
                </li>
            })}
        </ol>
    </>
}

function nameOf(code: string): string {
    return Countries.get(code)?.name ?? code
}
