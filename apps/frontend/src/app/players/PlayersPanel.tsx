import {useId} from "react"
import {PlayerLine, RosterEntry} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import {rosterGroups} from "../../domain/roster.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {UsersIcon} from "../components/icons.tsx"
import {truncate} from "../truncate.ts"
import "./Players.css"
import "./PlayerCard.css"

/** As long as the chat lets a name run, so a name is cut at the same letter in both. */
const NAME_MAX_LENGTH = 16

export type PlayersButtonProps = {
    entries: readonly RosterEntry[]
    onOpen: () => void
    buttonRef?: React.Ref<HTMLButtonElement>
}

/** One slab in the menu's actions: an icon and how many are playing. */
export function PlayersButton({entries, onOpen, buttonRef}: PlayersButtonProps) {
    const label = entries.length === 1 ? "1 player online" : `${entries.length} players online`
    return <button ref={buttonRef}
                   type="button"
                   className="button button-ghost menu-players"
                   aria-label={label}
                   title={label}
                   onClick={onOpen}>
        <UsersIcon size={24}/>
        <span className="menu-players-count">{entries.length}</span>
    </button>
}

export type PlayersPanelProps = {
    entries: readonly RosterEntry[]
    /** Absent, the names are plain text. */
    onOpenPlayer?: (player: PlayerLine) => void
}

export default function PlayersPanel({entries, onOpenPlayer}: PlayersPanelProps) {
    if (entries.length === 0) {
        return <p className="players-empty">Nobody is playing right now.</p>
    }

    const {players, guests} = rosterGroups(entries)
    return <div className="players-panel">
        {players.length > 0 && <PlayersGroup title="Players" entries={players} onOpenPlayer={onOpenPlayer}/>}
        {guests.length > 0 && <PlayersGroup title="Guests" entries={guests} onOpenPlayer={onOpenPlayer}/>}
    </div>
}

type PlayersGroupProps = {
    title: string
    entries: RosterEntry[]
    onOpenPlayer?: (player: PlayerLine) => void
}

function PlayersGroup({title, entries, onOpenPlayer}: PlayersGroupProps) {
    const titleId = useId()
    return <section className="players-group" aria-labelledby={titleId}>
        <h3 className="menu-section-title players-group-title" id={titleId}>
            {title}
            <span className="players-group-count">{entries.length}</span>
        </h3>
        <ul className="players-list">
            {entries.map((entry) => <li key={entry.key}
                                               className="players-entry"
                                               style={authorStyle(entry.name)}>
                <span className="players-entry-country"
                      role="img"
                      aria-label={countryName(entry.countryCode)}
                      title={countryName(entry.countryCode)}>
                    <CountryFlag code={entry.countryCode}/>
                </span>
                {onOpenPlayer
                    ? <button type="button"
                              className="players-entry-name player-name-button"
                              title={entry.name}
                              onClick={() => onOpenPlayer(entry)}>
                        {truncate(entry.name, NAME_MAX_LENGTH)}
                    </button>
                    : <span className="players-entry-name" title={entry.name}>
                        {truncate(entry.name, NAME_MAX_LENGTH)}
                    </span>}
                {entry.admin && <AdminCrown/>}
            </li>)}
        </ul>
    </section>
}

// The flag stands for the country, so its name is what the tooltip and the
// accessibility tree carry, as in the chat.
function countryName(code: string): string {
    return Countries.get(code)?.name ?? code
}
