import {useCallback, useEffect, useState} from "react"
import {loadPlaces, Places} from "../boards/places.ts"
import {Clients, CLIENTS} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {timeOf} from "../components/time.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {useCall} from "../useCall.ts"
import {SubmissionDetail} from "./SubmissionDetail.tsx"
import {SubmissionList} from "./SubmissionList.tsx"
import {Filter, Listed, listOf, nextOf, shownOf, waiting} from "./submissions.ts"

type Props = {
    // The submission open, by its id, or "" for none.
    opened: string
    onOpen: (submissionId: string) => void
    clients?: Clients
    // A board's places, by its name.
    places?: (board: string) => Promise<Places>
}

// What the operator did last, said above the next one.
type Done = {who: string, said: string}

// The submissions a verdict changed since the list was read, over that read.
type Changed = {on: Listed | undefined, submissions: Map<string, KeptSubmission>}

const FILTERS: {filter: Filter, label: string}[] = [
    {filter: "waiting", label: "Waiting"},
    {filter: "toOpen", label: "To open"},
    {filter: "all", label: "All"},
]

// An inbox: the flags players submitted on the left, the one open on the right. A verdict opens the next waiting one.
export function SubmissionsView({opened, onOpen, clients = CLIENTS, places: readPlaces = loadPlaces}: Props) {
    const {flags, player, planet} = clients
    const list = useCallback((signal: AbortSignal) => listOf(flags, player, signal), [flags, player])
    const [read, again] = useCall(list)
    // The lists as last read stay while they are read again.
    const [kept, setKept] = useState<Listed>()
    useEffect(() => {
        if (read.state === "done") setKept(read.value)
    }, [read])
    // The land around a pin is drawn on the board the world plays.
    const readGround = useCallback(
        async (signal: AbortSignal) => readPlaces((await planet.getTerritories({}, {signal})).board),
        [planet, readPlaces],
    )
    const [ground] = useCall(readGround)
    const [filter, setFilter] = useState<Filter>("waiting")
    const [changed, setChanged] = useState<Changed>({on: undefined, submissions: new Map()})
    const [last, setLast] = useState<Done>()

    const listed = read.state === "done" ? read.value : kept
    if (listed === undefined) {
        return read.state === "failed" ? <FailureNotice failure={read} onRetry={again}/> : <p className="muted">Loading…</p>
    }

    const submissions = listed.submissions.map(each => changed.on === listed ? changed.submissions.get(each.id) ?? each : each)
    const shown = shownOf(submissions, filter)
    const submission = submissions.find(each => each.id === opened)
    const count = (each: Filter) => shownOf(submissions, each).length

    const decided = (done: KeptSubmission, said: string) => {
        const before = submissions.filter(waiting)
        setChanged(was => ({
            on: listed,
            submissions: new Map([...(was.on === listed ? was.submissions : []), [done.id, done]]),
        }))
        setLast({who: done.name, said})
        onOpen(nextOf(before, done.id))
    }
    const openedFlag = (flag: Flag) => {
        const until = flag.sold?.endsAt?.toDate()
        setLast({who: flag.name, said: `Opened as ${flag.code}${until === undefined ? "" : `, until ${timeOf(until)}`}.`})
        again()
    }
    // A submission picked by hand: what the last verdict did is no longer the news.
    const pick = (submissionId: string) => {
        setLast(undefined)
        onOpen(submissionId)
    }
    const reload = () => {
        setLast(undefined)
        again()
    }

    return (
        <div className={opened === "" ? "inbox" : "inbox inbox-open"}>
            <section className="inbox-list" aria-label="Submissions">
                <div className="toolbar">
                    <div className="segmented" role="group" aria-label="Which submissions">
                        {FILTERS.map(each => (
                            <button
                                key={each.filter}
                                type="button"
                                aria-pressed={filter === each.filter}
                                onClick={() => setFilter(each.filter)}
                            >
                                {each.label} ({count(each.filter)})
                            </button>
                        ))}
                    </div>
                    <button type="button" className="button-small" disabled={read.state === "loading"} onClick={reload}>
                        Reload
                    </button>
                </div>
                {read.state === "failed" && <FailureNotice failure={read} onRetry={again}/>}
                {listed.names === undefined && <p className="muted">The founders' names could not be read.</p>}
                {shown.length === 0
                    ? <p className="muted">{emptyOf(filter)}</p>
                    : <SubmissionList submissions={shown} flags={listed.flags} names={listed.names} opened={opened} onOpen={pick}/>}
            </section>
            <div className="inbox-detail">
                {last !== undefined && (
                    <p className="done-banner" role="status"><strong>{last.who}</strong>: {last.said}</p>
                )}
                {submission !== undefined && (
                    <SubmissionDetail
                        key={submission.id}
                        submission={submission}
                        submissions={submissions}
                        flags={listed.flags}
                        names={listed.names}
                        places={ground}
                        clients={clients}
                        onBack={() => pick("")}
                        onOpen={pick}
                        onDecided={decided}
                        onOpened={openedFlag}
                    />
                )}
                {submission === undefined && opened !== "" && <p className="muted">This submission is not kept any more.</p>}
                {submission === undefined && opened === "" && <p className="muted inbox-pick">Pick a submission on the left.</p>}
            </div>
        </div>
    )
}

function emptyOf(filter: Filter): string {
    switch (filter) {
        case "waiting":
            return "No submission waits. All done."
        case "toOpen":
            return "No approved flag waits to be opened."
        case "all":
            return "No submission kept."
    }
}
