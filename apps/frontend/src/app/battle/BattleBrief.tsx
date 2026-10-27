import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {ClosedRound, Standing} from "../../backends/standings.ts"
import {BATTLE_BRIEF, briefStagesOf, TakesRow, takesRowsOf} from "../../domain/battleBrief.ts"
import {Countries} from "../../domain/countries.ts"
import {podiumOf, roundOverLine} from "../../domain/roundReveal.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {SwordsIcon} from "../components/icons.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {Podium} from "../season/RoundReveal.tsx"
import {PlaySound} from "../sound/soundPlayer.ts"
import TitleBadge from "../titles/TitleBadge.tsx"
import "../titles/TitleUnlocked.css"
import "../season/RoundReveal.css"
import "./Battle.css"

export type BattleBriefProps = {
    closed: ClosedRound
    description: string
    countryCode: string
    accountId?: string
    play: PlaySound
    onClose: () => void
}

export default function BattleBrief({closed, description, countryCode, accountId, play, onClose}: BattleBriefProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const [stages] = useState(() => briefStagesOf(closed))
    const [podium] = useState(() => podiumOf(closed))
    const [takes] = useState(() => takesRowsOf(closed, countryCode))
    const [at, setAt] = useState(0)
    const [ready, setReady] = useState(false)
    const stage = stages[at]
    const last = at === stages.length - 1

    useEffect(() => {
        play("title")
    }, [play])

    useEffect(() => {
        const timer = last
            ? setTimeout(() => setReady(true), BATTLE_BRIEF.settle * 1000)
            : setTimeout(() => setAt(at + 1), BATTLE_BRIEF[stage] * 1000)
        return () => clearTimeout(timer)
    }, [at, last, stage])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    return createPortal(
        <div ref={dialog}
             className="title-reveal round-reveal battle-brief"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={{"--steps": podium.length} as CSSProperties}>
            <div className="round-reveal-stage battle-brief-stage">
                <h2 id={headId} className="title-reveal-head round-reveal-head">{roundOverLine(closed)}</h2>
                {closed.times > 1 && <p className="battle-brief-times">
                    <SwordsIcon size={20}/>
                    <span>Points ×{closed.times}</span>
                </p>}
                {description && <p className="battle-brief-text">{description}</p>}

                {stage === "podium" && <Podium podium={podium}/>}
                {stage === "takes" && <Takes rows={takes}/>}
                {stage === "players" && <Players players={closed.players} accountId={accountId}/>}

                <div className="title-reveal-actions round-reveal-actions">
                    {!ready && !last &&
                        <button type="button" className="button" onClick={() => setAt(at + 1)}>Next</button>}
                    {ready && <button type="button" className="button button-gold" onClick={onClose}>Close</button>}
                </div>
            </div>
            {stage === "podium" && <div className="title-reveal-flash round-reveal-flash" aria-hidden="true"/>}
        </div>,
        document.body,
    )
}

function Takes({rows}: {rows: TakesRow[]}) {
    return <>
        <p className="round-reveal-season">Tiles taken</p>
        <ol className="battle-brief-list" aria-label="Tiles taken">
            {rows.map((row, i) => <li key={row.countryCode}
                                      className={row.mine ? "battle-brief-row battle-brief-row-mine" : "battle-brief-row"}
                                      style={{"--share": row.width, "--enter": i} as CSSProperties}>
                <CountryFlag code={row.countryCode}/>
                <span className="battle-brief-name">{nameOf(row.countryCode)}</span>
                <span className="battle-brief-tiles">{row.tiles}</span>
            </li>)}
        </ol>
    </>
}

function Players({players, accountId}: {players: Standing[], accountId?: string}) {
    return <>
        <p className="round-reveal-season">Top players</p>
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
