import {FlagsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {reopen, reopenRefusalOf} from "./actions.ts"

// A flag Claude rejected, reopened for the operator's own verdict: it waits again, as if its player had asked a person
// to look again.
export function ReopenPanel({flags, submission, onReopened}: {
    flags: FlagsAdmin, submission: KeptSubmission, onReopened: (reopened: KeptSubmission) => void,
}) {
    return (
        <section className="panel" aria-label="Look again">
            <h2>Look again</h2>
            <p className="hint">
                Claude rejected it. Reopen it to decide it yourself: it waits for your verdict, shown even unpaid, and its
                player can no longer ask a person to look again. What was paid was refunded: an approval waits to be paid
                for again.
            </p>
            <ConfirmAction
                label="Reopen"
                question={
                    <p>
                        Reopen <strong>{submission.name}</strong>? It waits for your verdict again, and its player sees it
                        waiting.
                    </p>
                }
                run={() => reopen(flags, submission.id)}
                said={() => "Reopened."}
                refusal={reopenRefusalOf}
                onDone={onReopened}
            />
        </section>
    )
}
