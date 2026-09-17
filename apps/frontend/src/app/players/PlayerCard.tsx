import {PlayerInfo, PlayerInfoBackend, RosterEntry} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {truncate} from "../truncate.ts"
import {usePlayerInfo} from "./usePlayerInfo.ts"
import "./PlayerCard.css"

/** Longer than the roster lets a name run: the card has the room. */
const NAME_MAX_LENGTH = 24

const day = new Intl.DateTimeFormat(undefined, {dateStyle: "medium"})
const count = new Intl.NumberFormat()

export type PlayerCardProps = {
    /** Who was clicked: a line of the roster, or the author of a chat message. */
    player: RosterEntry
    backend: PlayerInfoBackend
    onClose: () => void
}

export default function PlayerCard({player, backend, onClose}: PlayerCardProps) {
    const state = usePlayerInfo(backend, player)
    const country = Countries.get(player.countryCode)?.name ?? player.countryCode

    return <Modal title={truncate(player.name, NAME_MAX_LENGTH)} className="player-card" onClose={onClose}>
        <div className="player-card-who" style={authorStyle(player.name, player.tag)}>
            <span className="player-card-country" role="img" aria-label={country} title={country}>
                <CountryFlag code={player.countryCode}/>
            </span>
            <span className="player-card-country-name">{country}</span>
            <span className="player-card-tag">#{player.tag}</span>
        </div>

        {state.kind === "loading" && <p className="player-card-note" role="status">Loading…</p>}
        {state.kind === "guest" && <p className="player-card-note">
            Guests have no stats. Sign in and pick a username to get yours.
        </p>}
        {state.kind === "missing" && <p className="player-card-note">No player holds this name now.</p>}
        {state.kind === "failed" && <p className="player-card-note">The stats could not be loaded.</p>}
        {state.kind === "ready" && <PlayerStats info={state.info}/>}
    </Modal>
}

function PlayerStats({info}: {info: PlayerInfo}) {
    return <dl className="player-card-stats">
        <Stat label="Tiles taken" value={count.format(info.tilesTaken)}/>
        <Stat label="Streak" value={days(info.streakCurrent)}/>
        <Stat label="Best streak" value={days(info.streakBest)}/>
        {info.createdAt !== undefined && <Stat label="Playing since" value={day.format(info.createdAt)}/>}
    </dl>
}

function Stat({label, value}: {label: string, value: string}) {
    return <div className="player-card-stat">
        <dt>{label}</dt>
        <dd>{value}</dd>
    </div>
}

function days(n: number): string {
    return n === 1 ? "1 day" : `${count.format(n)} days`
}
