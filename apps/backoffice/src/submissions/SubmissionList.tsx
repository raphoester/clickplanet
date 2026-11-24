import {Ago} from "../components/When.tsx"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"
import {founderOf, looksLikePhoto, reasonLabel, stateLabel, toOpen} from "./submissions.ts"

type Props = {
    submissions: KeptSubmission[]
    names: Map<string, string> | undefined
    // The submissions whose founder's reply to their letter waits.
    replied: ReadonlySet<string>
    opened: string
    onOpen: (submissionId: string) => void
}

// One row a submission: its picture small, its name, new or a change, who sent it, how long ago, its verdict, and a
// reply that waits.
export function SubmissionList({submissions, names, replied, opened, onOpen}: Props) {
    return (
        <ol className="rows" aria-label="The submissions">
            {submissions.map(submission => (
                <li key={submission.id}>
                    <button
                        type="button"
                        className="row submission-row"
                        aria-current={submission.id === opened ? "true" : undefined}
                        onClick={() => onOpen(submission.id)}
                    >
                        {submission.picture === undefined
                            ? <span className="submission-thumb" aria-hidden="true"/>
                            : <img className="submission-thumb" src={submission.picture.url} alt=""/>}
                        <span className="submission-row-text">
                            <span className="row-head">
                                <span className="name">{submission.name}</span>
                                {looksLikePhoto(submission) && <span className="badge badge-report">Many colours</span>}
                                {replied.has(submission.id) && <span className="badge badge-replied">Replied</span>}
                                <span className="row-end"><Ago at={submission.submittedAt}/></span>
                            </span>
                            <span className="row-sub muted">
                                by {founderOf(submission.accountId, names)}
                                {submission.state !== SubmissionState.WAITING && ` · ${verdictOf(submission)}`}
                            </span>
                        </span>
                    </button>
                </li>
            ))}
        </ol>
    )
}

function verdictOf(submission: KeptSubmission): string {
    if (toOpen(submission)) return "Approved, to open"
    if (submission.rejection !== undefined) return `Rejected: ${reasonLabel(submission.rejection.reason)}`
    return stateLabel(submission.state)
}
