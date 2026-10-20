import {useCallback} from "react"
import {supportAdmin} from "../api.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Person} from "../components/Person.tsx"
import {When} from "../components/When.tsx"
import {Report} from "../gen/grpc/support/v1/admin_pb.ts"
import {countOf, reasonLabel} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {DecisionLine} from "./DecisionLine.tsx"

type Props = {
    openOnly: boolean
    onOpenOnly: (openOnly: boolean) => void
    onOpen: (report: Report) => void
}

export function ReportList({openOnly, onOpenOnly, onOpen}: Props) {
    const list = useCallback(
        (signal: AbortSignal) => supportAdmin.listReports({openOnly}, {signal}).then(answer => answer.reports),
        [openOnly],
    )
    const [reports, again] = useCall(list)

    return (
        <section aria-label="Reports">
            <div className="toolbar">
                <div className="segmented" role="group" aria-label="Which reports">
                    <button type="button" aria-pressed={openOnly} onClick={() => onOpenOnly(true)}>Open</button>
                    <button type="button" aria-pressed={!openOnly} onClick={() => onOpenOnly(false)}>All</button>
                </div>
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>

            {reports.state === "loading" && <p className="muted">Loading…</p>}
            {reports.state === "failed" && <FailureNotice failure={reports} onRetry={again}/>}
            {reports.state === "done" && (
                <>
                    <p className="muted">{countOf(reports.value.length, openOnly ? "open report" : "report")}</p>
                    <ol className="cards">
                        {reports.value.map(report => (
                            <ReportCard key={report.id} report={report} onOpen={() => onOpen(report)}/>
                        ))}
                    </ol>
                </>
            )}
        </section>
    )
}

function ReportCard({report, onOpen}: {report: Report, onOpen: () => void}) {
    return (
        <li className="card">
            <div className="card-head">
                <span className="badge badge-report">{reasonLabel(report.reason)}</span>
                <When at={report.createdAt}/>
            </div>
            <dl className="people">
                <Person label="Reported" name={report.reportedName} id={report.reportedAccountId}/>
                <Person label="By" name={report.reporterName} id={report.reporterAccountId}/>
            </dl>
            {report.text !== "" && <p className="text">{report.text}</p>}
            <div className="card-foot">
                {report.decision !== undefined ? <DecisionLine decision={report.decision}/> : <span/>}
                <button type="button" className="button" onClick={onOpen}>Open</button>
            </div>
        </li>
    )
}
