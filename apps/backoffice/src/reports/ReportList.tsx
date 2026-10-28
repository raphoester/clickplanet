import {useCallback} from "react"
import {ChatAdmin} from "../clients.ts"
import {Ago} from "../components/When.tsx"
import {Report, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {outcomeLabel, reasonLabel} from "../support/labels.ts"
import {useCall} from "../useCall.ts"

type Props = {
    chat: ChatAdmin
    reports: Report[]
    // How many reports, open or closed, name each account.
    counts: Map<string, number>
    opened: string
    onOpen: (reportId: string) => void
}

// One row a report: what it is about, who, how long ago, and what was said, so most are judged from the list.
export function ReportList({chat, reports, counts, opened, onOpen}: Props) {
    return (
        <ol className="rows" aria-label="The reports">
            {reports.map(report => (
                <li key={report.id}>
                    <button
                        type="button"
                        className="row"
                        aria-current={report.id === opened ? "true" : undefined}
                        onClick={() => onOpen(report.id)}
                    >
                        <span className="row-head">
                            <span className="badge badge-report">{reasonLabel(report.reason)}</span>
                            <span className="name">{report.reportedName === "" ? "no name" : report.reportedName}</span>
                            {(counts.get(report.reportedAccountId) ?? 0) > 1 && (
                                <span className="count" title="Reports on this account">
                                    ×{counts.get(report.reportedAccountId)}
                                </span>
                            )}
                            <span className="row-end"><Ago at={report.createdAt}/></span>
                        </span>
                        {report.messageId !== "" && report.decision === undefined
                            ? <ReportedText chat={chat} messageId={report.messageId} fallback={report.text}/>
                            : report.text !== "" && <span className="row-text">{report.text}</span>}
                        <span className="row-sub muted">
                            by {report.reporterName === "" ? "no name" : report.reporterName}
                            {report.decision !== undefined && (
                                <>
                                    {" · "}
                                    <span className={report.decision.outcome === ReportOutcome.ACTED ? "outcome-acted" : ""}>
                                        {outcomeLabel(report.decision.outcome)}
                                    </span>
                                    {report.decision.comment !== "" && `: ${report.decision.comment}`}
                                </>
                            )}
                        </span>
                    </button>
                </li>
            ))}
        </ol>
    )
}

// The reported line, read for each open report. What the reporter wrote stands in while it loads, or when it is gone.
function ReportedText({chat, messageId, fallback}: {chat: ChatAdmin, messageId: string, fallback: string}) {
    const read = useCallback((signal: AbortSignal) => chat.getMessage({messageId}, {signal}), [chat, messageId])
    const [message] = useCall(read)
    if (message.state !== "done") return fallback === "" ? null : <span className="row-text">{fallback}</span>
    const removed = message.value.removedAt !== undefined
    return <q className={removed ? "row-text line-removed" : "row-text"}>{message.value.text}</q>
}
