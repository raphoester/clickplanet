import {FormEvent, useCallback, useState} from "react"
import {PlanetAdmin, PlayerAdmin} from "../clients.ts"
import {CountryField} from "../components/CountryField.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {countryIdOf} from "../countries.ts"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Author} from "../gen/grpc/player/v1/internal_pb.ts"
import {Opened} from "../navigation.ts"
import {countOf} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {findPlayers, LIMITS, topPlayers, withAuthors} from "./actions.ts"
import {PlayerName} from "./PlayerName.tsx"
import {BanLine, PlayerStats} from "./PlayerStats.tsx"

export type Query =
    | {kind: "top", limit: number}
    | {kind: "find", flag: string, area: string, limit: number}

type Props = {
    planet: PlanetAdmin
    player: PlayerAdmin
    query: Query
    onQuery: (query: Query) => void
    onOpen: (opened: Opened) => void
}

// Who played in the ledger's 72 hours: the most takes over the whole map, or who painted one flag on one ground.
export function PlayerList({planet, player, query, onQuery, onOpen}: Props) {
    const [kind, setKind] = useState(query.kind)
    const [limit, setLimit] = useState(query.limit)
    const [flag, setFlag] = useState(query.kind === "find" ? query.flag : "")
    const [area, setArea] = useState(query.kind === "find" ? query.area : "")

    const showTop = (shown: number) => {
        setKind("top")
        setLimit(shown)
        onQuery({kind: "top", limit: shown})
    }

    const find = (event: FormEvent) => {
        event.preventDefault()
        onQuery({kind: "find", flag: countryIdOf(flag), area: countryIdOf(area), limit})
    }

    return (
        <section aria-label="Players" className="detail">
            <div className="toolbar">
                <div className="segmented" role="group" aria-label="Which players">
                    <button type="button" aria-pressed={kind === "top"} onClick={() => showTop(limit)}>Top</button>
                    <button type="button" aria-pressed={kind === "find"} onClick={() => setKind("find")}>By flag</button>
                </div>
                <label className="picker picker-inline">
                    <span className="muted">Show</span>
                    <select
                        value={limit}
                        onChange={event => kind === "top" ? showTop(Number(event.target.value)) : setLimit(Number(event.target.value))}
                    >
                        {LIMITS.map(n => <option key={n} value={n}>{n}</option>)}
                    </select>
                </label>
            </div>

            {kind === "find" && (
                <form className="panel" aria-label="Find players by flag" onSubmit={find}>
                    <p className="hint">Who took a tile for a flag, held or painted over since, latest take first.</p>
                    <div className="fields">
                        <CountryField label="Flag" value={flag} onChange={setFlag}/>
                        <CountryField label="On the ground of" value={area} onChange={setArea} empty="Empty: the whole map"/>
                    </div>
                    <button type="submit" className="button button-primary" disabled={countryIdOf(flag) === ""}>Find</button>
                </form>
            )}

            {query.kind === kind && <Results planet={planet} player={player} query={query} onOpen={onOpen}/>}
        </section>
    )
}

type ResultsProps = {planet: PlanetAdmin, player: PlayerAdmin, query: Query, onOpen: (opened: Opened) => void}

function Results({planet, player, query, onOpen}: ResultsProps) {
    const list = useCallback(
        (signal: AbortSignal) => (query.kind === "top"
            ? topPlayers(planet, query.limit, signal)
            : findPlayers(planet, query.flag, query.area, query.limit, signal)
        ).then(page => withAuthors(player, page, signal)),
        [planet, player, query],
    )
    const [players, again] = useCall(list)

    if (players.state === "loading") return <p className="muted">Loading…</p>
    if (players.state === "failed") return <FailureNotice failure={players} onRetry={again}/>

    const {players: shown, total, authors, unnamed} = players.value
    return (
        <>
            <div className="toolbar">
                <p className="muted">{shown.length < total ? `${shown.length} of ${total} players` : countOf(total, "player")}</p>
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>
            {unnamed !== undefined && <p className="action-refused">The names could not be read. {unnamed}</p>}
            <ol className="cards">
                {shown.map(row => (
                    <PlayerCard key={`${row.accountId}|${row.scope}`} row={row} author={authors.get(row.accountId)} onOpen={onOpen}/>
                ))}
            </ol>
        </>
    )
}

type CardProps = {row: Player, author: Author | undefined, onOpen: (opened: Opened) => void}

// A row by its flag and its name. The account id is on the player page, with the tools that take it.
function PlayerCard({row, author, onOpen}: CardProps) {
    const caller = row.accountId !== "" ? {accountId: row.accountId} : {scope: row.scope}
    const name = author?.name ?? ""

    return (
        <li className="card">
            <div className="card-head">
                <PlayerName name={name} flag={row.flagCountryId} unnamed={row.accountId === "" ? "No account" : "Unnamed"}/>
                <span className="muted">last take <When at={row.lastAt}/></span>
            </div>
            <div className="card-line">
                <span className="mono muted">{row.scope}</span>
                <BanLine player={row}/>
            </div>
            <div className="card-foot">
                <PlayerStats player={row}/>
                <button type="button" className="button" onClick={() => onOpen({caller, name, row})}>Open</button>
            </div>
        </li>
    )
}
