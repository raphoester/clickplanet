import {
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    SupportReportedAlreadyError,
} from "../../backends/support.ts"
import {Key} from "../../i18n/translator.ts"

export type SendFailure = "rate-limited" | "rejected" | "no-session" | "reported-already" | "failed"

export const SEND_FAILURES: Record<SendFailure, Key> = {
    "rate-limited": "support.failure.rateLimited",
    "rejected": "support.failure.rejected",
    "no-session": "support.failure.noSession",
    "reported-already": "support.failure.reportedAlready",
    "failed": "support.failure.failed",
}

export function sendFailureOf(e: unknown): SendFailure {
    if (e instanceof SupportRateLimitedError) return "rate-limited"
    if (e instanceof SupportRejectedError) return "rejected"
    if (e instanceof SupportNoSessionError) return "no-session"
    if (e instanceof SupportReportedAlreadyError) return "reported-already"
    return "failed"
}
