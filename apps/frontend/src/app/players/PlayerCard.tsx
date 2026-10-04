import {PlayerInfo, PlayerInfoBackend, PlayerLine, PlayerTitle} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {StatTile, StatTiles} from "../components/StatTiles.tsx"
import {days} from "../days.ts"
import OgStamp from "../titles/OgStamp.tsx"
import TitleBanner from "../titles/TitleBanner.tsx"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {metalOf, OG} from "../titles/titleArt.ts"
import {truncate} from "../truncate.ts"
import {usePlayerInfo} from "./usePlayerInfo.ts"
import "./PlayerCard.css"

const NAME_MAX_LENGTH = 24

const day = new Intl.DateTimeFormat(undefined, {dateStyle: "medium"})
const count = new Intl.NumberFormat()

export type PlayerCardProps = {
    player: PlayerLine
    backend: PlayerInfoBackend
    onClose: () => void
}

export default function PlayerCard({player, backend, onClose}: PlayerCardProps) {
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

    return <Modal title={title} className={`player-card${frame}`} onClose={onClose}>
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
