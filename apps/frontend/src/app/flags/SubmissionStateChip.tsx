import {SubmissionState} from "../../domain/flagSubmissions.ts"
import {t} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

const WORDS: Record<SubmissionState, Key> = {
    waiting: "flagSubmission.state.waiting",
    approved: "flagSubmission.state.approved",
    rejected: "flagSubmission.state.rejected",
    opened: "flagSubmission.state.opened",
}

export function SubmissionStateChip({state}: {state: SubmissionState}) {
    return <span className={`chip flag-state flag-state--${state}`}>{t(WORDS[state])}</span>
}
