import {FormEvent, useCallback, useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {AccountId, Copyable} from "../components/AccountId.tsx"
import {CountryField} from "../components/CountryField.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {countryIdOf} from "../countries.ts"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Opened} from "../navigation.ts"
import {countOf} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {findPlayers, LIMITS, topPlayers} from "./actions.ts"
import {BanLine, PlayerStats} from "./PlayerStats.tsx"

export type Query =
    | {kind: "top", limit: number}
    | {kind: "find", flag: string, area: string, limit: number}

type Props = {
    planet: PlanetAdmin
    query: Query
    onQuery: (query: Query) => void
    onOpen: (opened: Opened) => void
}

// Who played in the ledger's 72 hours: the most takes over the whole map, or who painted one flag on one ground.
export function PlayerList({planet, query, onQuery, onOpen}: Props) {
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

            {query.kind === kind && <Results planet={planet} query={query} onOpen={onOpen}/>}
        </section>
    )
}

function Results({planet, query, onOpen}: {planet: PlanetAdmin, query: Query, onOpen: (opened: Opened) => void}) {
    const list = useCallback(
        (signal: AbortSignal) => query.kind === "top"
            ? topPlayers(planet, query.limit, signal)
            : findPlayers(planet, query.flag, query.area, query.limit, signal),
        [planet, query],
    )
    const [players, again] = useCall(list)

    if (players.state === "loading") return <p className="muted">Loading…</p>
    if (players.state === "failed") return <FailureNotice failure={players} onRetry={again}/>

    const {players: shown, total} = players.value
    return (
        <>
            <div className="toolbar">
                <p className="muted">{shown.length < total ? `${shown.length} of ${total} players` : countOf(total, "player")}</p>
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>
            <ol className="cards">
                {shown.map(player => (
                    <PlayerCard key={`${player.accountId}|${player.scope}`} player={player} onOpen={onOpen}/>
                ))}
            </ol>
        </>
    )
}

function PlayerCard({player, onOpen}: {player: Player, onOpen: (opened: Opened) => void}) {
    const caller = player.accountId !== "" ? {accountId: player.accountId} : {scope: player.scope}

    return (
        <li className="card">
            <div className="card-head">
                <BanLine player={player}/>
                <span className="muted">last take <When at={player.lastAt}/></span>
            </div>
            <dl className="people">
                <div>
                    <dt>Account</dt>
                    <dd><AccountId id={player.accountId}/></dd>
                </div>
                <div>
                    <dt>Address</dt>
                    <dd><Copyable text={player.scope}/></dd>
                </div>
            </dl>
            <div className="card-foot">
                <PlayerStats player={player}/>
                <button type="button" className="button" onClick={() => onOpen({caller, row: player})}>Open</button>
            </div>
        </li>
    )
}
