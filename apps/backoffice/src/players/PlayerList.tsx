import {FormEvent, useCallback, useState} from "react"
import {PlanetAdmin, PlayerAdmin} from "../clients.ts"
import {CountryField} from "../components/CountryField.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Ago} from "../components/When.tsx"
import {countryIdOf, countryLabel} from "../countries.ts"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {PlayerMatch} from "../gen/grpc/player/v1/admin_pb.ts"
import {Author} from "../gen/grpc/player/v1/internal_pb.ts"
import {Opened} from "../navigation.ts"
import {countOf} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {findPlayers, LIMITS, searchPlayers, topPlayers, withAuthors} from "./actions.ts"
import {searchOf} from "./caller.ts"
import {spanOf} from "./durations.ts"
import {PlayerName} from "./PlayerName.tsx"

export type Query =
    | {kind: "top", limit: number}
    | {kind: "find", flag: string, area: string, limit: number}
    | NameQuery

type NameQuery = {kind: "name", name: string, limit: number}

type Props = {
    planet: PlanetAdmin
    player: PlayerAdmin
    query: Query
    onQuery: (query: Query) => void
    onOpen: (opened: Opened) => void
}

// One box finds a player by its name, its account id or its address; the list below is who played in the ledger's
// 72 hours, the whole map or one flag.
export function PlayerList({planet, player, query, onQuery, onOpen}: Props) {
    const [typed, setTyped] = useState(query.kind === "name" ? query.name : "")
    const [flag, setFlag] = useState(query.kind === "find" ? query.flag : "")
    const [area, setArea] = useState(query.kind === "find" ? query.area : "")
    const search = searchOf(typed)

    const find = (event: FormEvent) => {
        event.preventDefault()
        if (search === undefined) return
        if ("open" in search) onOpen({caller: search.open})
        else onQuery({kind: "name", name: search.name, limit: query.limit})
    }
    const filter = (event: FormEvent) => {
        event.preventDefault()
        onQuery({kind: "find", flag: countryIdOf(flag), area: countryIdOf(area), limit: query.limit})
    }
    const top = () => {
        setTyped("")
        setFlag("")
        setArea("")
        onQuery({kind: "top", limit: query.limit})
    }

    return (
        <section aria-label="Players" className="detail">
            <form className="search" role="search" aria-label="Find a player" onSubmit={find}>
                <input
                    type="search"
                    aria-label="Name, account id or address"
                    placeholder="Name, account id or address"
                    value={typed}
                    autoCapitalize="none"
                    autoComplete="off"
                    spellCheck={false}
                    onChange={event => setTyped(event.target.value)}
                />
                <button type="submit" className="button button-primary" disabled={search === undefined}>
                    {search !== undefined && "open" in search ? "Open" : "Find"}
                </button>
            </form>
            <form className="filters" aria-label="Who took tiles for a flag" onSubmit={filter}>
                <CountryField label="Flag" value={flag} onChange={setFlag} empty="Any"/>
                <CountryField label="On the ground of" value={area} onChange={setArea} empty="Anywhere"/>
                <button type="submit" className="button" disabled={countryIdOf(flag) === ""}>Filter</button>
                <label className="picker picker-inline">
                    <span className="muted">Show</span>
                    <select value={query.limit} onChange={event => onQuery({...query, limit: Number(event.target.value)})}>
                        {LIMITS.map(n => <option key={n} value={n}>{n}</option>)}
                    </select>
                </label>
            </form>
            <div className="toolbar">
                <h2 className="list-title">{titleOf(query)}</h2>
                {query.kind !== "top" && <button type="button" className="button-small" onClick={top}>× Back to the most active</button>}
            </div>
            {query.kind === "name"
                ? <NameResults player={player} query={query} onOpen={onOpen}/>
                : <Results planet={planet} player={player} query={query} onOpen={onOpen}/>}
        </section>
    )
}

function titleOf(query: Query): string {
    switch (query.kind) {
        case "top":
            return "Most active in the last 72 hours"
        case "find":
            return `Took tiles for ${countryLabel(query.flag)}${query.area === "" ? "" : ` on ${countryLabel(query.area)}'s ground`}`
        case "name":
            return `Names holding “${query.name}”`
    }
}

type Sort = {key: Column, down: boolean}
type Column = "takes" | "tiles" | "rate" | "active" | "last"

const COLUMNS: {key: Column, label: string, of: (row: Player) => number, wide?: boolean}[] = [
    {key: "takes", label: "Takes", of: row => row.takes},
    {key: "tiles", label: "Held", of: row => row.tiles},
    {key: "rate", label: "Takes/min", of: row => row.takesPerMinute, wide: true},
    {key: "active", label: "Active", of: row => Number(row.activeFor?.seconds ?? 0n), wide: true},
    {key: "last", label: "Last take", of: row => Number(row.lastAt?.seconds ?? 0n), wide: true},
]

type ResultsProps = {
    planet: PlanetAdmin
    player: PlayerAdmin
    query: Exclude<Query, NameQuery>
    onOpen: (opened: Opened) => void
}

function Results({planet, player, query, onOpen}: ResultsProps) {
    const list = useCallback(
        (signal: AbortSignal) => (query.kind === "top"
            ? topPlayers(planet, query.limit, signal)
            : findPlayers(planet, query.flag, query.area, query.limit, signal)
        ).then(page => withAuthors(player, page, signal)),
        [planet, player, query],
    )
    const [players, again] = useCall(list)
    // Unset is the server's order: most takes first, or the latest take first for a flag.
    const [sort, setSort] = useState<Sort | undefined>(undefined)

    if (players.state === "loading") return <p className="muted">Loading…</p>
    if (players.state === "failed") return <FailureNotice failure={players} onRetry={again}/>
    const {players: rows, total, authors, unnamed} = players.value
    const column = COLUMNS.find(each => each.key === sort?.key)
    const shown = column === undefined ? rows
        : [...rows].sort((a, b) => (column.of(a) - column.of(b)) * (sort?.down ? -1 : 1))
    const sortBy = (key: Column) => setSort(was => ({key, down: was?.key === key ? !was.down : true}))

    return (
        <>
            <div className="toolbar">
                <p className="muted">{rows.length < total ? `${rows.length} of ${total} players` : countOf(total, "player")}</p>
                <button type="button" className="button-small" onClick={again}>Reload</button>
            </div>
            {unnamed !== undefined && <p className="action-refused">The names could not be read. {unnamed}</p>}
            <table className="table players">
                <thead>
                    <tr>
                        <th>Player</th>
                        <th className="col-wide">Address</th>
                        {COLUMNS.map(each => (
                            <th
                                key={each.key}
                                className={each.wide ? "num col-wide" : "num"}
                                aria-sort={sort?.key === each.key ? (sort.down ? "descending" : "ascending") : undefined}
                            >
                                <button type="button" className="sort" onClick={() => sortBy(each.key)}>
                                    {each.label}{sort?.key === each.key && (sort.down ? " ▾" : " ▴")}
                                </button>
                            </th>
                        ))}
                        <th>Ban</th>
                    </tr>
                </thead>
                <tbody>
                    {shown.map(row => (
                        <PlayerRow key={`${row.accountId}|${row.scope}`} row={row} author={authors.get(row.accountId)} onOpen={onOpen}/>
                    ))}
                </tbody>
            </table>
        </>
    )
}

type RowProps = {row: Player, author: Author | undefined, onOpen: (opened: Opened) => void}

// A row by its flag and its name; the whole row opens it. The account id is on the player page, with the tools.
function PlayerRow({row, author, onOpen}: RowProps) {
    const caller = row.accountId !== "" ? {accountId: row.accountId} : {scope: row.scope}
    const name = author?.name ?? ""
    return (
        <tr className={row.banned === true ? "row-link row-banned" : "row-link"}>
            <td>
                <button type="button" className="cell-link" onClick={() => onOpen({caller, name, row})}>
                    <PlayerName name={name} flag={row.flagCountryId} unnamed={row.accountId === "" ? "No account" : "Unnamed"}/>
                </button>
            </td>
            <td className="mono muted col-wide">{row.scope}</td>
            <td className="num">{row.takes}</td>
            <td className="num">{row.tiles} <span className="muted">{shareOf(row)}</span></td>
            <td className="num col-wide">{row.takesPerMinute.toFixed(1)}</td>
            <td className="num col-wide">{spanOf(row.activeFor)}</td>
            <td className="num col-wide"><Ago at={row.lastAt}/></td>
            <td>{row.banned === true && <span className="status status-bad">Banned</span>}</td>
        </tr>
    )
}

// Many takes and few tiles held is a player painted over as fast as it paints.
function shareOf(row: Player): string {
    if (row.takes === 0) return ""
    const share = (100 * row.tiles) / row.takes
    return `${share < 10 ? share.toFixed(1) : Math.round(share)}%`
}

type NameResultsProps = {player: PlayerAdmin, query: NameQuery, onOpen: (opened: Opened) => void}

function NameResults({player, query, onOpen}: NameResultsProps) {
    const search = useCallback(
        (signal: AbortSignal) => searchPlayers(player, query.name, query.limit, signal),
        [player, query],
    )
    const [matches, again] = useCall(search)

    if (matches.state === "loading") return <p className="muted">Loading…</p>
    if (matches.state === "failed") return <FailureNotice failure={matches} onRetry={again}/>
    const {players, total} = matches.value
    if (players.length === 0) return <p className="muted">No account has a name holding “{query.name}”.</p>
    return (
        <>
            <p className="muted">{players.length < total ? `${players.length} of ${total} accounts` : countOf(total, "account")}</p>
            <table className="table players">
                <thead>
                    <tr><th>Name</th><th>Kind</th></tr>
                </thead>
                <tbody>
                    {players.map(match => <MatchRow key={match.accountId} match={match} onOpen={onOpen}/>)}
                </tbody>
            </table>
        </>
    )
}

// An account by its name alone: the ledger is not asked, so it may have taken nothing in its 72 hours.
function MatchRow({match, onOpen}: {match: PlayerMatch, onOpen: (opened: Opened) => void}) {
    return (
        <tr className="row-link">
            <td>
                <button
                    type="button"
                    className="cell-link"
                    onClick={() => onOpen({caller: {accountId: match.accountId}, name: match.name})}
                >
                    <PlayerName name={match.name} flag="" unnamed="Unnamed"/>
                </button>
            </td>
            <td className="muted">{match.guest ? "guest code" : "username"}</td>
        </tr>
    )
}
