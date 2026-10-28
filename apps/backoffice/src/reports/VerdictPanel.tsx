import {useState} from "react"
import {Clients} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {Report, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {DEFAULT_MUTE_SECONDS, LADDER} from "../players/actions.ts"
import {BAN_CHOICES, Choice, MUTE_CHOICES} from "../players/durations.ts"
import {MAX_COMMENT_LENGTH} from "./actions.ts"
import {actsOf, close, Closed, haltedWords, questionOf, Verdict} from "./verdict.ts"

type Props = {
    clients: Clients
    report: Report
    who: string
    // Whether the reported message is shown and not removed yet.
    removable: boolean
    // The other open reports on the same account, which the verdict may close too.
    others: Report[]
    onClosed: (closed: Closed) => void
}

// What to do to the reported account and the report, picked together and sent with one question.
export function VerdictPanel({clients, report, who, removable, others, onClosed}: Props) {
    const [remove, setRemove] = useState(false)
    const [muting, setMuting] = useState(false)
    const [muteFor, setMuteFor] = useState(DEFAULT_MUTE_SECONDS)
    const [banning, setBanning] = useState(false)
    const [banFor, setBanFor] = useState(LADDER)
    const [closeOthers, setCloseOthers] = useState(true)
    const [comment, setComment] = useState("")

    const account = report.reportedAccountId
    const verdict: Verdict = {
        remove: remove && removable,
        muteFor: muting && account !== "" ? muteFor : undefined,
        banFor: banning && account !== "" ? banFor : undefined,
        comment,
    }
    const acts = actsOf(verdict, who)
    const reportIds = [report.id, ...(closeOthers ? others.map(other => other.id) : [])]
    const target = {accountId: account, messageId: report.messageId, who}
    const run = (outcome: ReportOutcome) => () => close(clients, target, outcome === ReportOutcome.ACTED
        ? verdict
        : {remove: false, muteFor: undefined, banFor: undefined, comment}, outcome, reportIds)

    return (
        <section className="panel verdict" aria-label="Verdict">
            <h2>{report.decision === undefined ? "Verdict" : "Change the decision"}</h2>
            <div className="verdict-acts">
                {report.messageId !== "" && (
                    <label className="check">
                        <input
                            type="checkbox"
                            checked={verdict.remove}
                            disabled={!removable}
                            onChange={event => setRemove(event.target.checked)}
                        />
                        {removable ? "Remove the message" : "Remove the message (gone already)"}
                    </label>
                )}
                {account !== "" && (
                    <>
                        <Timed label="Mute" on={muting} onOn={setMuting} choices={MUTE_CHOICES} value={muteFor} onValue={setMuteFor}/>
                        <Timed label="Ban" on={banning} onOn={setBanning} choices={BAN_CHOICES} value={banFor} onValue={setBanFor}/>
                    </>
                )}
            </div>
            {others.length > 0 && (
                <label className="check">
                    <input type="checkbox" checked={closeOthers} onChange={event => setCloseOthers(event.target.checked)}/>
                    Close the {others.length === 1 ? "other open report" : `${others.length} other open reports`} on {who} too
                </label>
            )}
            <label className="field">
                <span className="muted">Comment (optional)</span>
                <textarea
                    value={comment}
                    maxLength={MAX_COMMENT_LENGTH}
                    rows={2}
                    onChange={event => setComment(event.target.value)}
                />
            </label>
            <div className="verdict-buttons">
                <ConfirmAction
                    label={acts.length === 0 ? "Close as acted" : "Act and close"}
                    tone={verdict.banFor !== undefined ? "danger" : undefined}
                    question={questionOf(acts, reportIds.length, ReportOutcome.ACTED)}
                    run={run(ReportOutcome.ACTED)}
                    said={closed => closed.said.join(" ")}
                    refusal={haltedWords}
                    onDone={onClosed}
                />
                <ConfirmAction
                    label="Dismiss"
                    question={questionOf([], reportIds.length, ReportOutcome.DISMISSED)}
                    run={run(ReportOutcome.DISMISSED)}
                    said={closed => closed.said.join(" ")}
                    refusal={haltedWords}
                    onDone={onClosed}
                />
            </div>
        </section>
    )
}

type TimedProps = {
    label: string
    on: boolean
    onOn: (on: boolean) => void
    choices: Choice[]
    value: number
    onValue: (seconds: number) => void
}

// A step that takes a duration: its box, and the duration beside it.
function Timed({label, on, onOn, choices, value, onValue}: TimedProps) {
    return (
        <div className="check-row">
            <label className="check">
                <input type="checkbox" checked={on} onChange={event => onOn(event.target.checked)}/>
                {label}
            </label>
            <select
                aria-label={`${label} for`}
                value={value}
                onChange={event => {
                    onValue(Number(event.target.value))
                    onOn(true)
                }}
            >
                {choices.map(choice => <option key={choice.seconds} value={choice.seconds}>{choice.label}</option>)}
            </select>
        </div>
    )
}
