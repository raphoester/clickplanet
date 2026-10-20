import {When} from "../components/When.tsx"
import {ReportDecision, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {outcomeLabel} from "../support/labels.ts"

export function DecisionLine({decision}: {decision: ReportDecision}) {
    return (
        <div className="decision">
            <span className={`badge ${decision.outcome === ReportOutcome.ACTED ? "badge-acted" : "badge-dismissed"}`}>
                {outcomeLabel(decision.outcome)}
            </span>
            <When at={decision.decidedAt}/>
            {decision.comment !== "" && <p className="text decision-comment">{decision.comment}</p>}
        </div>
    )
}
