import {ShownState} from "../../domain/flagSubmissions.ts"
import {t} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

// A submission as its player reads it, or a flag held that the operator removed.
export type ChipState = ShownState | "removed"

const WORDS: Record<ChipState, Key> = {
    waiting: "flagSubmission.state.waiting",
    approved: "flagSubmission.state.approved",
    rejected: "flagSubmission.state.rejected",
    opened: "flagSubmission.state.opened",
    closed: "flagSubmission.state.closed",
    removed: "flagSubmission.state.removed",
}

export function SubmissionStateChip({state}: {state: ChipState}) {
    return <span className={`chip flag-state flag-state--${state}`}>{t(WORDS[state])}</span>
}
