import {supportAdmin} from "../api.ts"
import {AccountId} from "../components/AccountId.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {Report} from "../gen/grpc/support/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {countOf, reasonLabel} from "./labels.ts"

const list = (signal: AbortSignal) => supportAdmin.listReports({}, {signal}).then(answer => answer.reports)

export function ReportList() {
    const [reports, again] = useCall(list)

    return (
        <section aria-label="Reports">
            <div className="toolbar">
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>

            {reports.state === "loading" && <p className="muted">Loading…</p>}
            {reports.state === "failed" && <FailureNotice failure={reports} onRetry={again}/>}
            {reports.state === "done" && (
                <>
                    <p className="muted">{countOf(reports.value.length, "report")}</p>
                    <ol className="cards">
                        {reports.value.map(report => <ReportCard key={report.id} report={report}/>)}
                    </ol>
                </>
            )}
        </section>
    )
}

function ReportCard({report}: {report: Report}) {
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
            {report.messageId !== "" && (
                <p className="muted">Message <code className="mono">{report.messageId}</code></p>
            )}
        </li>
    )
}

function Person({label, name, id}: {label: string, name: string, id: string}) {
    return (
        <div>
            <dt>{label}</dt>
            <dd>
                <span className={name === "" ? "muted" : "name"}>{name === "" ? "no name" : name}</span>
                <AccountId id={id}/>
            </dd>
        </div>
    )
}
