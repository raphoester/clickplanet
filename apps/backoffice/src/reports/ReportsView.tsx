import {Timestamp} from "@bufbuild/protobuf"
import {useCallback, useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Report, ReportDecision} from "../gen/grpc/support/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {ReportDetail} from "./ReportDetail.tsx"
import {ReportList} from "./ReportList.tsx"
import {Closed} from "./verdict.ts"

// The server's most: every report is read once, so the page counts each account's reports, open or closed.
export const MAX_REPORTS = 500

type Props = {
    // The report open, by its id, or "" for none.
    opened: string
    onOpen: (reportId: string) => void
    clients?: Clients
}

// An inbox: the reports on the left, the one open on the right. A verdict closes it and opens the next open one.
export function ReportsView({opened, onOpen, clients = CLIENTS}: Props) {
    const [openOnly, setOpenOnly] = useState(true)
    const list = useCallback(
        (signal: AbortSignal) => clients.support.listReports({limit: MAX_REPORTS}, {signal}).then(answer => answer.reports),
        [clients.support],
    )
    const [call, again] = useCall(list)
    // The decisions made on this page since the list was read.
    const [decided, setDecided] = useState(new Map<string, ReportDecision>())
    const [last, setLast] = useState<Closed | undefined>(undefined)

    if (call.state === "loading") return <p className="muted">Loading…</p>
    if (call.state === "failed") return <FailureNotice failure={call} onRetry={again}/>

    const reports = call.value.map(report => decided.has(report.id) ? new Report({...report, decision: decided.get(report.id)}) : report)
    const open = reports.filter(report => report.decision === undefined)
    const shown = openOnly ? open : reports
    const report = reports.find(each => each.id === opened)
    const counts = new Map<string, number>()
    for (const each of reports) counts.set(each.reportedAccountId, (counts.get(each.reportedAccountId) ?? 0) + 1)

    const closed = (done: Closed) => {
        const decision = new ReportDecision({outcome: done.outcome, comment: done.comment, decidedAt: Timestamp.fromDate(done.decidedAt)})
        setDecided(was => new Map([...was, ...done.reportIds.map(id => [id, decision] as const)]))
        setLast(done)
        // The next open report below the one closed, or else the first one left.
        const left = open.filter(each => !done.reportIds.includes(each.id))
        const at = open.findIndex(each => each.id === opened)
        onOpen((left.find(each => open.indexOf(each) > at) ?? left[0])?.id ?? "")
    }
    // A report picked by hand: what the last verdict did is no longer the news.
    const pick = (reportId: string) => {
        setLast(undefined)
        onOpen(reportId)
    }
    const reload = () => {
        setDecided(new Map())
        setLast(undefined)
        again()
    }

    return (
        <div className={opened === "" ? "inbox" : "inbox inbox-open"}>
            <section className="inbox-list" aria-label="Reports">
                <div className="toolbar">
                    <div className="segmented" role="group" aria-label="Which reports">
                        <button type="button" aria-pressed={openOnly} onClick={() => setOpenOnly(true)}>Open ({open.length})</button>
                        <button type="button" aria-pressed={!openOnly} onClick={() => setOpenOnly(false)}>All ({reports.length})</button>
                    </div>
                    <button type="button" className="button-small" onClick={reload}>Reload</button>
                </div>
                {shown.length === 0
                    ? <p className="muted">{openOnly ? "No open report. All done." : "No report kept."}</p>
                    : <ReportList chat={clients.chat} reports={shown} counts={counts} opened={opened} onOpen={pick}/>}
            </section>
            <div className="inbox-detail">
                {last !== undefined && (
                    <p className="done-banner" role="status">
                        <strong>{last.who}</strong>: {last.said.join(" ")}
                    </p>
                )}
                {report !== undefined && (
                    <ReportDetail
                        key={report.id}
                        report={report}
                        related={reports.filter(each => each.id !== report.id && each.reportedAccountId === report.reportedAccountId
                            && report.reportedAccountId !== "")}
                        clients={clients}
                        onBack={() => pick("")}
                        onOpen={pick}
                        onClosed={closed}
                    />
                )}
                {report === undefined && opened !== "" && <p className="muted">This report is not kept any more.</p>}
                {report === undefined && opened === "" && <p className="muted inbox-pick">Pick a report on the left.</p>}
            </div>
        </div>
    )
}
