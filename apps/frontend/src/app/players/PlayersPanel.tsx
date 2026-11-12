import {useId} from "react"
import {RosterEntry} from "../../backends/player.ts"
import {rosterGroups} from "../../domain/roster.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import Streak from "../badges/Streak.tsx"
import WornTitle from "../badges/WornTitle.tsx"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {usePlayerCard} from "./playerCardStore.ts"
import {rosterEntries, useRoster} from "./rosterStore.ts"
import {t, translator} from "../../i18n/page.ts"
import "./Players.css"
import "./PlayerCard.css"

const NAME_MAX_LENGTH = 16

export default function PlayersPanel() {
    const entries = useRoster(rosterEntries)
    if (entries.length === 0) {
        return <p className="players-empty">{t("player.nobodyPlaying")}</p>
    }

    const {players, guests} = rosterGroups(entries)
    return <div className="players-panel">
        {players.length > 0 && <PlayersGroup title={t("player.players")} entries={players}/>}
        {guests.length > 0 && <PlayersGroup title={t("player.guests")} entries={guests}/>}
    </div>
}

type PlayersGroupProps = {
    title: string
    entries: RosterEntry[]
}

function PlayersGroup({title, entries}: PlayersGroupProps) {
    const titleId = useId()
    const words = useWordFilter()
    const openPlayer = usePlayerCard((card) => card.open)
    return <section className="players-group" aria-labelledby={titleId}>
        <h3 className="menu-section-title players-group-title" id={titleId}>
            {title}
            <span className="players-group-count">{translator().number(entries.length)}</span>
        </h3>
        <ul className="players-list">
            {entries.map((entry) => <li key={entry.key}
                                               className="players-entry"
                                               style={authorStyle(entry)}>
                <span className="players-entry-country"
                      role="img"
                      aria-label={translator().country(entry.countryCode)}
                      title={translator().country(entry.countryCode)}>
                    <CountryFlag code={entry.countryCode}/>
                </span>
                <button type="button"
                        className="players-entry-name player-name-button"
                        title={words.masked(entry.name)}
                        onClick={() => openPlayer(entry)}>
                    {truncate(words.masked(entry.name), NAME_MAX_LENGTH)}
                </button>
                {entry.admin && <AdminCrown/>}
                <WornTitle accountId={entry.accountId} size={18}/>
                <Streak accountId={entry.accountId}/>
            </li>)}
        </ul>
    </section>
}
