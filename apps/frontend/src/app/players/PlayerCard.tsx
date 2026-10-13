import {useState} from "react"
import {PlayerInfo, PlayerInfoBackend, PlayerLine, PlayerTitle} from "../../backends/player.ts"
import {PlayerReporter} from "../../backends/support.ts"
import {Countries} from "../../domain/countries.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {StatTile, StatTiles} from "../components/StatTiles.tsx"
import {days} from "../days.ts"
import ReportPlayerModal from "../support/ReportPlayerModal.tsx"
import OgStamp from "../titles/OgStamp.tsx"
import TitleBanner from "../titles/TitleBanner.tsx"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {metalOf, OG} from "../titles/titleArt.ts"
import {truncate} from "../truncate.ts"
import PlayerFronts from "./PlayerFronts.tsx"
import {usePlayerInfo} from "./usePlayerInfo.ts"
import "./PlayerCard.css"

const NAME_MAX_LENGTH = 24

const day = new Intl.DateTimeFormat(undefined, {dateStyle: "medium"})
const count = new Intl.NumberFormat()

export type PlayerCardProps = {
    player: PlayerLine
    backend: PlayerInfoBackend
    // With it the card offers to report the player; `messageId` is the chat message the card was opened from.
    reporter?: PlayerReporter
    messageId?: string
    onClose: () => void
}

export default function PlayerCard({player, backend, reporter, messageId, onClose}: PlayerCardProps) {
    const [reporting, setReporting] = useState(false)
    const state = usePlayerInfo(backend, player)
    const country = Countries.get(player.countryCode)?.name ?? player.countryCode

    const admin = player.admin || (state.kind === "ready" && state.info.admin)
    const color = state.kind === "ready" ? state.info.color : player.color
    const info = state.kind === "ready" ? state.info : undefined
    const title = <span className="player-card-title" style={authorStyle({...player, color})}>
        {truncate(player.name, NAME_MAX_LENGTH)}
        {admin && <AdminCrown size={20}/>}
        {info?.titles.some((held) => held.id === OG) && <OgStamp/>}
    </span>
    const frame = info?.wornTitle ? ` title-frame-${metalOf(info.wornTitle)}` : ""

    if (reporting && reporter) {
        return <ReportPlayerModal accountId={player.accountId}
                                  name={player.name}
                                  messageId={messageId}
                                  reporter={reporter}
                                  onClose={() => setReporting(false)}/>
    }

    const report = reporter && player.accountId !== "" && <button type="button"
                                                                  className="player-card-report"
                                                                  onClick={() => setReporting(true)}>
        Report
    </button>

    return <Modal title={title} className={`player-card${frame}`} footer={report} onClose={onClose}>
        <div className="player-card-who">
            <span className="player-card-country" role="img" aria-label={country} title={country}>
                <CountryFlag code={player.countryCode}/>
            </span>
            <span className="player-card-country-name">{country}</span>
        </div>

        {state.kind === "loading" && <p className="player-card-note" role="status">Loading…</p>}
        {state.kind === "missing" && <p className="player-card-note">No player holds this name now.</p>}
        {state.kind === "failed" && <p className="player-card-note">The stats could not be loaded.</p>}
        {info?.wornTitle && <TitleBanner title={info.wornTitle}/>}
        {info && <PlayerTitles titles={info.titles}/>}
        {info && <PlayerStats info={info}/>}
        {info && <PlayerFronts playsFor={info.playsFor} playsAgainst={info.playsAgainst}/>}
    </Modal>
}

function PlayerTitles({titles}: {titles: PlayerTitle[]}) {
    if (titles.length === 0) return null

    return <ul className="player-card-titles" aria-label="Titles">
        {titles.map((title) => <li key={title.id} className="player-card-held">
            <TitleEmblem title={title} size={48} ribbon/>
            <span className="player-card-held-name">{title.name}</span>
        </li>)}
    </ul>
}

export function PlayerStats({info}: {info: PlayerInfo}) {
    return <StatTiles>
        <StatTile label="Tiles taken" value={count.format(info.tilesTaken)}/>
        <StatTile label="Streak" value={days(info.streakCurrent)}/>
        <StatTile label="Best streak" value={days(info.streakBest)}/>
        {info.createdAt !== undefined && <StatTile label="Playing since" value={day.format(info.createdAt)}/>}
    </StatTiles>
}
