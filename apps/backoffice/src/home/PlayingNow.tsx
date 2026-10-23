import {Country} from "../components/Country.tsx"
import {flagOf} from "../countries.ts"
import {RosterEntry} from "../gen/grpc/player/v1/player_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {wholeOf} from "./activity.ts"

const FLAGS = 6

export function PlayingNow({roster}: {roster: RosterEntry[]}) {
    const openPlayer = useOpenPlayer()
    const named = roster.filter(entry => !entry.guest)
    const guests = roster.length - named.length
    const flags = flagsOf(roster)

    return (
        <section className="panel" aria-label="Playing now">
            <div className="panel-head">
                <h2>Playing now</h2>
                <span className="muted">{wholeOf(named.length)} signed in, {wholeOf(guests)} {guests === 1 ? "guest" : "guests"}</span>
            </div>
            {roster.length === 0 && <p className="muted">Nobody is playing.</p>}
            {named.length > 0 && (
                <ul className="chips">
                    {named.map(entry => (
                        <li key={entry.key}>
                            {openPlayer !== undefined && entry.accountId !== ""
                                ? (
                                    <button type="button" className="chip"
                                            onClick={() => openPlayer({caller: {accountId: entry.accountId}, name: entry.name})}>
                                        <FlagOf id={entry.countryId}/>{entry.name}
                                    </button>
                                )
                                : <span className="chip"><FlagOf id={entry.countryId}/>{entry.name}</span>}
                        </li>
                    ))}
                </ul>
            )}
            {flags.length > 0 && (
                <ul className="flag-counts" aria-label="Playing for">
                    {flags.map(([country, count]) => (
                        <li key={country}><Country id={country}/> <b>{count}</b></li>
                    ))}
                </ul>
            )}
        </section>
    )
}

function FlagOf({id}: {id: string}) {
    const flag = flagOf(id)
    return flag === "" ? null : <span className="flag" aria-hidden="true">{flag} </span>
}

function flagsOf(roster: RosterEntry[]): [string, number][] {
    const counts = new Map<string, number>()
    for (const entry of roster) counts.set(entry.countryId, (counts.get(entry.countryId) ?? 0) + 1)
    return [...counts].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).slice(0, FLAGS)
}
