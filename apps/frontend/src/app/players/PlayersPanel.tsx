import {useId} from "react"
import {RosterEntry} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import {rosterGroups} from "../../domain/roster.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import StreakFlame from "../components/StreakFlame.tsx"
import TitleBadge from "../titles/TitleBadge.tsx"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {usePlayerCard} from "./playerCardStore.ts"
import {rosterEntries, useRoster} from "./rosterStore.ts"
import "./Players.css"
import "./PlayerCard.css"

const NAME_MAX_LENGTH = 16

export default function PlayersPanel() {
    const entries = useRoster(rosterEntries)
    if (entries.length === 0) {
        return <p className="players-empty">Nobody is playing right now.</p>
    }

    const {players, guests} = rosterGroups(entries)
    return <div className="players-panel">
        {players.length > 0 && <PlayersGroup title="Players" entries={players}/>}
        {guests.length > 0 && <PlayersGroup title="Guests" entries={guests}/>}
    </div>
}

type PlayersGroupProps = {
    title: string
    entries: RosterEntry[]
}

function PlayersGroup({title, entries}: PlayersGroupProps) {
    const titleId = useId()
    const words = useWordFilter()
    const openPlayer = usePlayerCard((card) => card.openable ? card.open : undefined)
    return <section className="players-group" aria-labelledby={titleId}>
        <h3 className="menu-section-title players-group-title" id={titleId}>
            {title}
            <span className="players-group-count">{entries.length}</span>
        </h3>
        <ul className="players-list">
            {entries.map((entry) => <li key={entry.key}
                                               className="players-entry"
                                               style={authorStyle(entry)}>
                <span className="players-entry-country"
                      role="img"
                      aria-label={countryName(entry.countryCode)}
                      title={countryName(entry.countryCode)}>
                    <CountryFlag code={entry.countryCode}/>
                </span>
                {openPlayer
                    ? <button type="button"
                              className="players-entry-name player-name-button"
                              title={words.masked(entry.name)}
                              onClick={() => openPlayer(entry)}>
                        {truncate(words.masked(entry.name), NAME_MAX_LENGTH)}
                    </button>
                    : <span className="players-entry-name" title={words.masked(entry.name)}>
                        {truncate(words.masked(entry.name), NAME_MAX_LENGTH)}
                    </span>}
                {entry.admin && <AdminCrown/>}
                <TitleBadge title={entry.wornTitle} size={18}/>
                <StreakFlame days={entry.streak}/>
            </li>)}
        </ul>
    </section>
}

function countryName(code: string): string {
    return Countries.get(code)?.name ?? code
}
