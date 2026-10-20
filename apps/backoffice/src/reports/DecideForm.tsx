import {useState} from "react"
import {SupportAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {timeOf} from "../components/time.ts"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {outcomeLabel} from "../support/labels.ts"
import {decide, Decided, MAX_COMMENT_LENGTH, refusalOf} from "./actions.ts"

const OUTCOMES = [ReportOutcome.ACTED, ReportOutcome.DISMISSED]

type Props = {
    support: SupportAdmin
    reportId: string
    decided: boolean
    onDecided: (decided: Decided) => void
}

export function DecideForm({support, reportId, decided, onDecided}: Props) {
    const [outcome, setOutcome] = useState<ReportOutcome | undefined>(undefined)
    const [comment, setComment] = useState("")

    return (
        <section className="panel" aria-label="Close the report">
            <h2>{decided ? "Change the decision" : "Close the report"}</h2>
            <div className="segmented" role="radiogroup" aria-label="Outcome">
                {OUTCOMES.map(value => (
                    <button
                        key={value}
                        type="button"
                        role="radio"
                        aria-checked={outcome === value}
                        onClick={() => setOutcome(value)}
                    >
                        {outcomeLabel(value)}
                    </button>
                ))}
            </div>
            <label className="field">
                <span className="muted">Comment (optional)</span>
                <textarea
                    value={comment}
                    maxLength={MAX_COMMENT_LENGTH}
                    rows={3}
                    onChange={event => setComment(event.target.value)}
                />
            </label>
            <ConfirmAction
                label="Close the report"
                disabled={outcome === undefined}
                question={`Close this report as ${outcomeLabel(outcome ?? ReportOutcome.UNSPECIFIED)}?`}
                run={() => decide(support, reportId, outcome ?? ReportOutcome.UNSPECIFIED, comment)}
                said={done => `Closed as ${outcomeLabel(done.outcome)} at ${timeOf(done.decidedAt)}.`}
                refusal={err => refusalOf("decide", err)}
                onDone={onDecided}
            />
        </section>
    )
}
