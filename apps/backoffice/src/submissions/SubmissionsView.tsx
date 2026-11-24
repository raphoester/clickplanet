import {useCallback, useEffect, useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {timeOf} from "../components/time.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {useCall} from "../useCall.ts"
import {SubmissionDetail} from "./SubmissionDetail.tsx"
import {SubmissionList} from "./SubmissionList.tsx"
import {Filter, Ground, loadGround, Listed, listOf, nextOf, repliedOf, shownOf, waiting} from "./submissions.ts"

type Props = {
    // The submission open, by its id, or "" for none.
    opened: string
    onOpen: (submissionId: string) => void
    clients?: Clients
    // A board's land and capitals, by its name.
    ground?: (board: string) => Promise<Ground>
}

// What the operator did last, said above the next one.
type Done = {who: string, said: string}

// The submissions a verdict changed since the list was read, over that read.
type Changed = {on: Listed | undefined, submissions: Map<string, KeptSubmission>}

const FILTERS: {filter: Filter, label: string}[] = [
    {filter: "waiting", label: "Waiting"},
    {filter: "toOpen", label: "To open"},
    {filter: "replies", label: "Replies"},
    {filter: "all", label: "All"},
]

const NO_REPLIES: ReadonlySet<string> = new Set()

// An inbox: the flags players submitted on the left, the one open on the right. A verdict opens the next waiting one.
export function SubmissionsView({opened, onOpen, clients = CLIENTS, ground: readBoard = loadGround}: Props) {
    const {flags, player, planet, support} = clients
    const list = useCallback((signal: AbortSignal) => listOf(flags, player, signal), [flags, player])
    const [read, again] = useCall(list)
    // The lists as last read stay while they are read again.
    const [kept, setKept] = useState<Listed>()
    useEffect(() => {
        if (read.state === "done") setKept(read.value)
    }, [read])
    // A capital is drawn on the board the world plays.
    const readGround = useCallback(
        async (signal: AbortSignal) => readBoard((await planet.getTerritories({}, {signal})).board),
        [planet, readBoard],
    )
    const [ground] = useCall(readGround)
    const readReplied = useCallback((signal: AbortSignal) => repliedOf(support, signal), [support])
    const [repliedCall, readRepliedAgain] = useCall(readReplied)
    const [keptReplied, setKeptReplied] = useState<ReadonlySet<string>>(NO_REPLIES)
    useEffect(() => {
        if (repliedCall.state === "done") setKeptReplied(repliedCall.value)
    }, [repliedCall])
    const [filter, setFilter] = useState<Filter>("waiting")
    const [changed, setChanged] = useState<Changed>({on: undefined, submissions: new Map()})
    const [last, setLast] = useState<Done>()

    const listed = read.state === "done" ? read.value : kept
    if (listed === undefined) {
        return read.state === "failed" ? <FailureNotice failure={read} onRetry={again}/> : <p className="muted">Loading…</p>
    }

    const submissions = listed.submissions.map(each => changed.on === listed ? changed.submissions.get(each.id) ?? each : each)
    const replied = repliedCall.state === "done" ? repliedCall.value : keptReplied
    const shown = shownOf(submissions, filter, replied)
    const submission = submissions.find(each => each.id === opened)
    const count = (each: Filter) => shownOf(submissions, each, replied).length

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
        readRepliedAgain()
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
                {repliedCall.state === "failed" && <p className="muted">The letters' replies could not be read.</p>}
                {shown.length === 0
                    ? <p className="muted">{emptyOf(filter)}</p>
                    : (
                        <SubmissionList
                            submissions={shown}
                            names={listed.names}
                            replied={replied}
                            opened={opened}
                            onOpen={pick}
                        />
                    )}
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
                        ground={ground}
                        clients={clients}
                        onBack={() => pick("")}
                        onOpen={pick}
                        onDecided={decided}
                        onOpened={openedFlag}
                        onLettersChanged={readRepliedAgain}
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
        case "replies":
            return "No player's reply waits."
        case "all":
            return "No submission kept."
    }
}
