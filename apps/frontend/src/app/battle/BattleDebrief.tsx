import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import PlayerName from "../nameplates/PlayerName.tsx"
import {createPortal} from "react-dom"
import {ClosedRound, Standing} from "../../backends/standings.ts"
import {BATTLE_DEBRIEF, DebriefScreen, PointsRow, pointsRowsOf} from "../../domain/battleBrief.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {SwordsIcon} from "../components/icons.tsx"
import SwordsArt from "../components/SwordsArt.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import WornTitle from "../badges/WornTitle.tsx"
import "../titles/TitleUnlocked.css"
import "../season/RoundReveal.css"
import "./Battle.css"

export type BattleDebriefProps = {
    screen: DebriefScreen
    closed: ClosedRound
    countryCode: string
    accountId?: string
    // Another screen follows this one.
    more: boolean
    play: PlaySound
    onClose: () => void
}

export default function BattleDebrief({screen, closed, countryCode, accountId, more, play, onClose}: BattleDebriefProps) {
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
                {screen === "over" && <Over headId={headId} closed={closed}/>}
                {screen === "points" && <Points headId={headId} closed={closed} rows={pointsRowsOf(closed, countryCode)}/>}
                {screen === "players" && <Players headId={headId} players={closed.players} accountId={accountId}/>}

                <div className="title-reveal-actions round-reveal-actions">
                    {ready && <button type="button" className={more ? "button" : "button button-gold"} onClick={onClose}>
                        {more ? t("common.next") : t("battle.toTheGlobe")}
                    </button>}
                </div>
            </div>
            {screen === "over" && <div className="title-reveal-flash" aria-hidden="true"/>}
        </div>,
        document.body,
    )
}

function Over({headId, closed}: {headId: string, closed: ClosedRound}) {
    return <>
        <h2 id={headId} className="title-reveal-head round-reveal-head">{t(closed.finale ? "finale.finished" : "battle.finished")}</h2>
        <div className="title-reveal-medal">
            <div className="title-reveal-rays" aria-hidden="true"/>
            <div className="title-reveal-wave" aria-hidden="true"/>
            <span className="title-reveal-float">
                <span className="battle-reveal-disc" aria-hidden="true"><SwordsArt/></span>
            </span>
        </div>
    </>
}

function Points({headId, closed, rows}: {headId: string, closed: ClosedRound, rows: PointsRow[]}) {
    return <>
        <h2 id={headId} className="title-reveal-head round-reveal-head">{t("battle.pointsWon")}</h2>
        {closed.times > 1 && <p className="battle-brief-times">
            <SwordsIcon size={20}/>
            <span>{t("battle.pointsTimes", {times: closed.times})}</span>
        </p>}
        <ol className="battle-brief-list" aria-label={t("battle.pointsWon")}>
            {rows.map((row, i) => <li key={row.countryCode}
                                      className={row.mine ? "battle-brief-row battle-brief-row-mine" : "battle-brief-row"}
                                      style={{"--share": row.width, "--enter": i} as CSSProperties}
                                      aria-current={row.mine ? "true" : undefined}>
                <span className="round-reveal-rank">{row.rank}</span>
                <CountryFlag code={row.countryCode}/>
                <span className="battle-brief-name">{translator().country(row.countryCode)}</span>
                {closed.times > 1 && <span className="battle-brief-times-of">{t("battle.wonTimes", {points: row.won, times: closed.times})}</span>}
                <span className="battle-brief-tiles">{translator().number(row.points)}</span>
            </li>)}
        </ol>
    </>
}

function Players({headId, players, accountId}: {headId: string, players: Standing[], accountId?: string}) {

    return <>
        <h2 id={headId} className="title-reveal-head round-reveal-head">{t("standings.topPlayers")}</h2>
        <ol className="battle-brief-list" aria-label={t("standings.topPlayers")}>
            {players.map((player, i) => {
                const you = player.accountId === accountId
                return <li key={player.accountId}
                           className={you ? "battle-brief-row battle-brief-row-mine" : "battle-brief-row"}
                           style={{"--enter": i} as CSSProperties}
                           aria-current={you ? "true" : undefined}>
                    <span className="round-reveal-rank">{player.rank}</span>
                    <span className="battle-brief-flag" role="img" aria-label={translator().country(player.countryCode)}>
                        <CountryFlag code={player.countryCode}/>
                    </span>
                    <PlayerName accountId={player.accountId} className="battle-brief-name battle-brief-player"/>
                    <WornTitle accountId={player.accountId} size={18}/>
                    <span className="battle-brief-tiles">{translator().number(player.tiles)}</span>
                </li>
            })}
        </ol>
    </>
}
