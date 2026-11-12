import {useCallback, useEffect, useId, useRef, useState} from "react"
import {PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import {PlayerTitle} from "../../backends/progression.ts"
import {PlayerReporter} from "../../backends/support.ts"
import {useBadge, useBadges} from "../badges/badgesStore.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import Bubble, {BUBBLE_MS} from "../components/Bubble.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {StatTile, StatTiles} from "../components/StatTiles.tsx"
import ReportPlayerModal from "../support/ReportPlayerModal.tsx"
import OgStamp from "../titles/OgStamp.tsx"
import TitleBanner from "../titles/TitleBanner.tsx"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {metalOf, OG} from "../titles/titleArt.ts"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import PlayerFronts from "./PlayerFronts.tsx"
import {usePlayerInfo} from "./usePlayerInfo.ts"
import {t, translator} from "../../i18n/page.ts"
import "./PlayerCard.css"

const NAME_MAX_LENGTH = 24

export type PlayerCardProps = {
    player: PlayerLine
    backend: PlayerInfoBackend
    // With it the card offers to report the player.
    reporter?: PlayerReporter
    onClose: () => void
}

export default function PlayerCard({player, backend, reporter, onClose}: PlayerCardProps) {
    const [reporting, setReporting] = useState(false)
    const state = usePlayerInfo(backend, player)
    const badge = useBadge(player.accountId)
    const words = useWordFilter()
    const country = translator().country(player.countryCode)

    const admin = player.admin || (state.kind === "ready" && state.info.admin)
    const color = state.kind === "ready" ? state.info.color : player.color
    const info = state.kind === "ready" ? state.info : undefined
    const title = <span className="player-card-title" style={authorStyle({...player, color})}>
        {truncate(words.masked(player.name), NAME_MAX_LENGTH)}
        {admin && <AdminCrown size={20}/>}
        {info?.titles.some((held) => held.id === OG) && <OgStamp/>}
    </span>
    // The card reads the player afresh, and its badge with it, so the title worn agrees with the titles held.
    useEffect(() => useBadges.getState().expire(player.accountId), [player.accountId])
    const worn = info && badge?.wornTitle
    const frame = worn ? ` title-frame-${metalOf(worn)}` : ""

    if (reporting && reporter) {
        return <ReportPlayerModal accountId={player.accountId}
                                  name={player.name}
                                  reporter={reporter}
                                  onClose={() => setReporting(false)}/>
    }

    const report = reporter && player.accountId !== "" && <button type="button"
                                                                  className="player-card-report"
                                                                  onClick={() => setReporting(true)}>
        {t("player.report")}
    </button>

    return <Modal title={title} className={`player-card${frame}`} footer={report} onClose={onClose}>
        <div className="player-card-who">
            <span className="player-card-country" role="img" aria-label={country} title={country}>
                <CountryFlag code={player.countryCode}/>
            </span>
            <span className="player-card-country-name">{country}</span>
        </div>

        {state.kind === "loading" && <p className="player-card-note" role="status">{t("common.loading")}</p>}
        {state.kind === "missing" && <p className="player-card-note">{t("player.gone")}</p>}
        {state.kind === "failed" && <p className="player-card-note">{t("player.statsFailed")}</p>}
        {worn && <TitleBanner title={worn}/>}
        {info && <PlayerTitles titles={info.titles}/>}
        {info && <PlayerStats info={info}/>}
        {info && <PlayerFronts playsFor={info.playsFor} playsAgainst={info.playsAgainst}/>}
    </Modal>
}

function PlayerTitles({titles}: {titles: PlayerTitle[]}) {
    if (titles.length === 0) return null

    return <ul className="player-card-titles" aria-label={t("player.titles")}>
        {titles.map((title) => <li key={title.id}><HeldTitle title={title}/></li>)}
    </ul>
}

type Shown = "pressed" | "hovered"

function HeldTitle({title}: {title: PlayerTitle}) {
    const button = useRef<HTMLButtonElement>(null)
    const described = useId()
    const [shown, setShown] = useState<Shown | undefined>()
    const hide = useCallback(() => setShown(undefined), [])

    useEffect(() => {
        if (shown !== "pressed") return
        const timer = setTimeout(hide, BUBBLE_MS)
        return () => clearTimeout(timer)
    }, [shown, hide])

    const medal = <>
        <TitleEmblem title={title} size={48} ribbon/>
        <span className="player-card-held-name">{title.name}</span>
    </>
    if (!title.feat) return <div className="player-card-held">{medal}</div>

    return <>
        <button type="button"
                ref={button}
                className="player-card-held"
                aria-describedby={described}
                onClick={() => setShown((now) => now === "pressed" ? undefined : "pressed")}
                onPointerEnter={(event) => event.pointerType === "mouse" && setShown((now) => now ?? "hovered")}
                onPointerLeave={(event) => event.pointerType === "mouse" && setShown((now) => now === "hovered" ? undefined : now)}>
            {medal}
            <span id={described} hidden>{title.feat}</span>
        </button>
        {shown && <Bubble anchor={button} onLost={hide}>{title.feat}</Bubble>}
    </>
}

export function PlayerStats({info}: {info: PlayerInfo}) {
    return <StatTiles>
        <StatTile label={t("player.tilesTaken")} value={translator().number(info.tilesTaken)}/>
        <StatTile label={t("player.streak")} value={t("common.days", {count: info.streakCurrent})}/>
        <StatTile label={t("player.bestStreak")} value={t("common.days", {count: info.streakBest})}/>
        {info.createdAt !== undefined && <StatTile label={t("player.playingSince")}
                                                   value={translator().dateTime(info.createdAt, {dateStyle: "medium"})}/>}
    </StatTiles>
}
