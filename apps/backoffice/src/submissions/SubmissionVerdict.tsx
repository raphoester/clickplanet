import {useState} from "react"
import {FlagsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {RejectionReason} from "../gen/grpc/flags/v1/flags_pb.ts"
import {approve, approveRefusalOf, lengthOf, MAX_NOTE, reject, rejectionReady, rejectRefusalOf} from "./actions.ts"
import {reasonLabel, REASONS} from "./submissions.ts"

type Props = {
    flags: FlagsAdmin
    submission: KeptSubmission
    // What the flag a change is for is named now.
    changes: string | undefined
    onDecided: (decided: KeptSubmission, said: string) => void
}

// One verdict: Approve, or Reject with the reason its player reads. Each asks first.
export function SubmissionVerdict({flags, submission, changes, onDecided}: Props) {
    const [reason, setReason] = useState<RejectionReason>()
    const [note, setNote] = useState("")
    const {id, name} = submission
    const noted = lengthOf(note) > 0

    return (
        <section className="panel verdict" aria-label="Verdict">
            <h2>Verdict</h2>
            <ConfirmAction
                label="Approve"
                question={changes === undefined
                    ? <p>Approve <strong>{name}</strong>? Its picture becomes public, and the flag waits to be opened.</p>
                    : <p>Approve the new name and picture of <strong>{changes}</strong>? Every player sees them at once.</p>}
                run={() => approve(flags, id)}
                said={() => "Approved."}
                refusal={approveRefusalOf}
                onDone={decided => onDecided(decided, changes === undefined ? "Approved." : `Approved: ${changes} is ${decided.name} now.`)}
            />
            <div className="reject">
                <label className="field">
                    <span className="muted">Reason</span>
                    <select
                        value={reason ?? ""}
                        onChange={event => setReason(event.target.value === "" ? undefined : Number(event.target.value) as RejectionReason)}
                    >
                        <option value="">Pick a reason</option>
                        {REASONS.map(each => <option key={each.reason} value={each.reason}>{each.label}</option>)}
                    </select>
                </label>
                <label className="field">
                    <span className="muted">
                        Note{reason === RejectionReason.OTHER ? "" : " (optional)"} · {lengthOf(note)}/{MAX_NOTE}
                    </span>
                    <textarea rows={3} value={note} onChange={event => setNote(event.target.value)}/>
                </label>
                <p className="hint">The player reads the reason and the note.</p>
                {lengthOf(note) > MAX_NOTE && <p className="action-refused" role="alert">A note of {MAX_NOTE} characters at most.</p>}
                <ConfirmAction
                    label="Reject"
                    tone="danger"
                    disabled={!rejectionReady(reason, note)}
                    question={reason !== undefined && (
                        <p>
                            Reject <strong>{name}</strong> as “{reasonLabel(reason)}”? The player reads this reason
                            {noted ? " and your note" : ""}.
                        </p>
                    )}
                    run={() => reject(flags, id, reason ?? RejectionReason.UNSPECIFIED, note)}
                    said={() => "Rejected."}
                    refusal={rejectRefusalOf}
                    onDone={decided => onDecided(decided, `Rejected as “${reasonLabel(reason ?? RejectionReason.UNSPECIFIED)}”.`)}
                />
            </div>
        </section>
    )
}
