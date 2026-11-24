import {useState} from "react"
import {FlagsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {localOf, timeOf, zoneOf} from "../components/time.ts"
import {daysLater, endOf, leftOf} from "../flags/actions.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {openRefusalOf, openSubmitted} from "./actions.ts"

const FIRST_DAYS = 30

// An approved new flag, opened for every player once paid for, until the end picked, at its submission's capital.
export function OpenSubmittedPanel({flags, submission, onOpened}: {
    flags: FlagsAdmin, submission: KeptSubmission, onOpened: (flag: Flag) => void,
}) {
    const [endsAt, setEndsAt] = useState(() => localOf(daysLater(new Date(), FIRST_DAYS)))
    const now = new Date()
    const end = endOf(endsAt)
    const gone = end !== undefined && end <= now

    return (
        <section className="panel" aria-label="Open the flag">
            <h2>Open the flag</h2>
            <label className="field open-end">
                <span className="muted">Ends</span>
                <input type="datetime-local" value={endsAt} onChange={event => setEndsAt(event.target.value)}/>
            </label>
            <p className="hint">Times are in {zoneOf()}.{end !== undefined && !gone && ` ${leftOf(end, now)}.`}</p>
            {gone && <p className="action-refused" role="alert">An end in the future.</p>}
            <ConfirmAction
                label="Open"
                disabled={end === undefined || gone}
                question={end !== undefined && (
                    <>
                        <p>Open <strong>{submission.name}</strong> until {timeOf(end)} ({leftOf(end, now)}), at its capital?</p>
                        <p>
                            Every player sees it at once and may play for it. A flag is never deleted: closed, it stays in
                            the list for good.
                        </p>
                    </>
                )}
                run={() => openSubmitted(flags, submission.id, end ?? now)}
                said={flag => `Opened ${flag.name} (${flag.code}).`}
                refusal={openRefusalOf}
                onDone={onOpened}
            />
        </section>
    )
}
