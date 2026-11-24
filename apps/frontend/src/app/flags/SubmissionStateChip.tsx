import {ShownState} from "../../domain/flagSubmissions.ts"
import {t} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

const WORDS: Record<ShownState, Key> = {
    waiting: "flagSubmission.state.waiting",
    approved: "flagSubmission.state.approved",
    rejected: "flagSubmission.state.rejected",
    opened: "flagSubmission.state.opened",
    closed: "flagSubmission.state.closed",
}

export function SubmissionStateChip({state}: {state: ShownState}) {
    return <span className={`chip flag-state flag-state--${state}`}>{t(WORDS[state])}</span>
}
