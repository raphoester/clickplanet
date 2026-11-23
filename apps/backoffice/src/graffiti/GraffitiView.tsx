import {useCallback, useState} from "react"
import {loadPlaces, Places} from "../boards/places.ts"
import {Clients, CLIENTS} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Ago} from "../components/When.tsx"
import {blankFlag, flagOf} from "../countries.ts"
import {FlagMark} from "../components/FlagMark.tsx"
import {Graffiti} from "../gen/grpc/vandalsquad/v1/admin_pb.ts"
import {Reading} from "../players/actions.ts"
import {Call, useCall} from "../useCall.ts"
import {GraffitiDetail} from "./GraffitiDetail.tsx"
import {GraffitiPicture} from "./GraffitiPicture.tsx"
import {bannedOf, boardsOf, keyOf, listOf, nameOf, paintersOf, placesByBoard, standingsOf, tilesOf} from "./graffiti.ts"

type Props = {
    // The graffiti open, by its key, or "" for none.
    opened: string
    onOpen: (key: string) => void
    clients?: Clients
    // A board's places, by the name a graffiti gives.
    places?: (board: string) => Promise<Places>
}

const LOADING: Call<Places> = {state: "loading"}

// The swastikas the vandal squad flagged, drawn: the list on the left, the one open on the right, with the ban and
// the revert of its painter. The squad flags by rules, so each is for a person to look at before acting.
export function GraffitiView({opened, onOpen, clients = CLIENTS, places: readPlaces = loadPlaces}: Props) {
    const {vandalsquad, player, antibot} = clients
    const list = useCallback((signal: AbortSignal) => listOf(vandalsquad, player, signal), [vandalsquad, player])
    const [call, again] = useCall(list)
    // Each graffiti is drawn on its own board, each board read once.
    const boards = call.state === "done" && call.value.on ? boardsOf(call.value.graffiti).join("\n") : ""
    const readBoards = useCallback(
        () => placesByBoard(boards === "" ? [] : boards.split("\n"), readPlaces),
        [boards, readPlaces],
    )
    const [boardsCall, readBoardsAgain] = useCall(readBoards)
    const painters = call.state === "done" && call.value.on ? paintersOf(call.value.graffiti).join(",") : ""
    const readStandings = useCallback(
        (signal: AbortSignal) => standingsOf(antibot, painters === "" ? [] : painters.split(","), signal),
        [antibot, painters],
    )
    const [standingsCall, readStandingsAgain] = useCall(readStandings)
    const [unbannedOnly, setUnbannedOnly] = useState(false)
    // The list, and where each painter stands now: a painter may have been banned from another page meanwhile.
    const reload = () => {
        again()
        readStandingsAgain()
    }

    if (call.state === "loading") return <p className="muted">Loading…</p>
    if (call.state === "failed") return <FailureNotice failure={call} onRetry={again}/>
    if (!call.value.on) return <p className="muted">The vandal squad is off on this server: nothing is flagged.</p>

    const {graffiti, names} = call.value
    const standings = standingsCall.state === "done" ? standingsCall.value : new Map<string, Reading>()
    const unbanned = graffiti.filter(each => bannedOf(standings.get(each.accountId)) !== true)
    const shown = unbannedOnly ? unbanned : graffiti
    const open = graffiti.find(each => keyOf(each) === opened)
    const placesOf = (each: Graffiti): Call<Places> =>
        boardsCall.state === "done" ? boardsCall.value.get(each.board) ?? LOADING : boardsCall.state === "failed" ? boardsCall : LOADING
    const unread = boardsCall.state === "done" ? [...boardsCall.value].filter(([, places]) => places.state === "failed") : []

    return (
        <div className={opened === "" ? "inbox" : "inbox inbox-open"}>
            <section className="inbox-list" aria-label="Flagged graffiti">
                <p className="hint">
                    Each drawing the vandal squad read as a swastika on the map. It flags by rules: look at each before
                    acting on its painter.
                </p>
                <div className="toolbar">
                    <div className="segmented" role="group" aria-label="Which graffiti">
                        <button type="button" aria-pressed={!unbannedOnly} onClick={() => setUnbannedOnly(false)}>
                            All ({graffiti.length})
                        </button>
                        <button type="button" aria-pressed={unbannedOnly} onClick={() => setUnbannedOnly(true)}>
                            Painter not banned ({standingsCall.state === "done" ? unbanned.length : "…"})
                        </button>
                    </div>
                    <button type="button" className="button-small" onClick={reload}>Reload</button>
                </div>
                {names === undefined && <p className="muted">The painters' names could not be read.</p>}
                {unread.map(([board, places]) => places.state === "failed" && (
                    <FailureNotice key={board} failure={places} onRetry={readBoardsAgain}/>
                ))}
                {shown.length === 0
                    ? <p className="muted">{unbannedOnly ? "Every painter is banned." : "Nothing flagged."}</p>
                    : (
                        <GraffitiList
                            graffiti={shown}
                            all={graffiti}
                            names={names}
                            standings={standings}
                            placesOf={placesOf}
                            opened={opened}
                            onOpen={onOpen}
                        />
                    )}
            </section>
            <div className="inbox-detail">
                {open !== undefined && (
                    <GraffitiDetail
                        key={opened}
                        graffiti={open}
                        name={names?.get(open.accountId) ?? ""}
                        others={graffiti.filter(each => each !== open && each.accountId === open.accountId && open.accountId !== "")}
                        places={placesOf(open)}
                        clients={clients}
                        onBack={() => onOpen("")}
                        onOpen={onOpen}
                        onBanned={readStandingsAgain}
                    />
                )}
                {open === undefined && opened !== "" && <p className="muted">This graffiti is not kept any more.</p>}
                {open === undefined && opened === "" && <p className="muted inbox-pick">Pick a graffiti on the left.</p>}
            </div>
        </div>
    )
}

type ListProps = {
    graffiti: Graffiti[]
    // Every graffiti read, to count each painter's.
    all: Graffiti[]
    names: Map<string, string> | undefined
    standings: Map<string, Reading>
    // The places of a graffiti's board.
    placesOf: (graffiti: Graffiti) => Call<Places>
    opened: string
    onOpen: (key: string) => void
}

// One row a graffiti: the drawing small, who painted it with which flag, how long ago, and whether it is banned now.
function GraffitiList({graffiti, all, names, standings, placesOf, opened, onOpen}: ListProps) {
    const counts = new Map<string, number>()
    for (const each of all) counts.set(each.accountId, (counts.get(each.accountId) ?? 0) + 1)

    return (
        <ol className="rows" aria-label="The graffiti">
            {graffiti.map(each => {
                const key = keyOf(each)
                const name = nameOf(each.accountId, names)
                const banned = bannedOf(standings.get(each.accountId))
                const count = counts.get(each.accountId) ?? 0
                const places = placesOf(each)
                return (
                    <li key={key}>
                        <button
                            type="button"
                            className="row graffiti-row"
                            aria-current={key === opened ? "true" : undefined}
                            onClick={() => onOpen(key)}
                        >
                            {places.state === "done"
                                ? <GraffitiPicture places={places.value} tileIds={each.tileIds} label={`The drawing ${name} painted`} small/>
                                : <span className="graffiti-thumb" aria-hidden="true"/>}
                            <span className="graffiti-row-text">
                                <span className="row-head">
                                    {blankFlag(each.countryId)
                                        ? <FlagMark id={each.countryId}/>
                                        : <span className="flag" aria-hidden="true">{flagOf(each.countryId)}</span>}
                                    <span className={each.accountId === "" ? "muted" : "name"}>{name}</span>
                                    {count > 1 && each.accountId !== "" && (
                                        <span className="count" title="Graffiti by this account">×{count}</span>
                                    )}
                                    <span className="row-end"><Ago at={each.flaggedAt}/></span>
                                </span>
                                <span className="row-sub muted">
                                    {tilesOf(each)}
                                    {banned === true && <>{" · "}<span className="status-bad-text">banned</span></>}
                                    {banned === false && " · not banned"}
                                </span>
                            </span>
                        </button>
                    </li>
                )
            })}
        </ol>
    )
}
