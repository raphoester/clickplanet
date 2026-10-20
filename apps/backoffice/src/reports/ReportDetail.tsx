import {Timestamp} from "@bufbuild/protobuf"
import {useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {Person} from "../components/Person.tsx"
import {When} from "../components/When.tsx"
import {Report, ReportDecision} from "../gen/grpc/support/v1/admin_pb.ts"
import {reasonLabel} from "../support/labels.ts"
import {decide, Decided} from "./actions.ts"
import {ActPanel} from "./ActPanel.tsx"
import {DecideForm} from "./DecideForm.tsx"
import {DecisionLine} from "./DecisionLine.tsx"
import {MessagePanel} from "./MessagePanel.tsx"
import {TheirMessages} from "./TheirMessages.tsx"

type Props = {
    report: Report
    onBack: () => void
    clients?: Clients
}

export function ReportDetail({report, onBack, clients = CLIENTS}: Props) {
    const [decision, setDecision] = useState<ReportDecision | undefined>(report.decision)
    const parties = {
        reporter: {accountId: report.reporterAccountId, name: report.reporterName},
        reported: {accountId: report.reportedAccountId, name: report.reportedName},
    }

    const decided = (done: Decided) => setDecision(new ReportDecision({
        outcome: done.outcome,
        comment: done.comment,
        decidedAt: Timestamp.fromDate(done.decidedAt),
    }))

    return (
        <section className="detail" aria-label="Report">
            <button type="button" className="button-small" onClick={onBack}>← Reports</button>

            <article className="card">
                <div className="card-head">
                    <span className="badge badge-report">{reasonLabel(report.reason)}</span>
                    <When at={report.createdAt}/>
                </div>
                <dl className="people">
                    <Person label="Reported" name={report.reportedName} id={report.reportedAccountId}/>
                    <Person label="By" name={report.reporterName} id={report.reporterAccountId}/>
                </dl>
                {report.text !== "" ? <p className="text">{report.text}</p> : <p className="muted">No details.</p>}
                {decision !== undefined && <DecisionLine decision={decision}/>}
            </article>

            {report.messageId !== "" && (
                <MessagePanel chat={clients.chat} messageId={report.messageId} parties={parties}/>
            )}

            {report.reportedAccountId !== "" && (
                <TheirMessages
                    chat={clients.chat}
                    accountId={report.reportedAccountId}
                    name={report.reportedName}
                    parties={parties}
                />
            )}

            {report.reportedAccountId !== "" && (
                <ActPanel
                    chat={clients.chat}
                    planet={clients.planet}
                    accountId={report.reportedAccountId}
                    name={report.reportedName}
                />
            )}

            <DecideForm
                decide={(outcome, comment) => decide(clients.support, report.id, outcome, comment)}
                decided={decision !== undefined}
                onDecided={decided}
            />
        </section>
    )
}
