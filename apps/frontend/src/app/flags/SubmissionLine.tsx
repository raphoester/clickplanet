import {RejectionReason, Shown} from "../../domain/flagSubmissions.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {Key} from "../../i18n/translator.ts"
import {useFlags} from "./flagsStore.ts"

const REASONS: Record<RejectionReason, Key> = {
    "not-a-place": "flagSubmission.rejected.notAPlace",
    "exists": "flagSubmission.rejected.exists",
    "unreadable": "flagSubmission.rejected.unreadable",
    "hateful": "flagSubmission.rejected.hateful",
    "other": "flagSubmission.rejected.other",
}

function dateOf(ms: number): string {
    return translator().dateTime(ms, {dateStyle: "medium"})
}

export type SubmissionLineProps = {
    shown: Shown
    // While the shop sells, a flag not paid for is paid on its paywall.
    selling: boolean
}

// Where a submission is, as its player reads it.
export default function SubmissionLine({shown, selling}: SubmissionLineProps) {
    const catalog = useFlags((store) => store.catalog)
    const flag = <strong>{shown.name}</strong>
    const sold = catalog.soldOf(shown.flag)
    switch (shown.state) {
        case "waiting":
            if (shown.paid) return t("flagSubmission.status.paid")
            return selling ? t("flagSubmission.status.checkAfterPay") : t("flagSubmission.status.waiting", {date: dateOf(shown.submittedAt)})
        case "approved":
            if (shown.paid) return t("flagSubmission.status.opening")
            return selling ? t("flagSubmission.status.pay") : t("flagSubmission.status.approved")
        case "rejected":
            return t(REASONS[shown.rejection?.reason ?? "other"])
        case "opened":
            return sold === undefined
                ? rich("flagSubmission.status.opened", {flag})
                : rich("flagSubmission.status.openUntil", {flag, date: dateOf(sold.endsAt)})
        case "closed":
            return sold?.closedAt === undefined ? undefined : rich("flagSubmission.status.closed", {flag, date: dateOf(sold.closedAt)})
    }
}
