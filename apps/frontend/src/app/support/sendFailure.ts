import {SupportNoSessionError, SupportRateLimitedError, SupportRejectedError} from "../../backends/support.ts"

export type SendFailure = "rate-limited" | "rejected" | "no-session" | "failed"

export const SEND_FAILURES: Record<SendFailure, string> = {
    "rate-limited": "You sent too many. Try again in a few minutes.",
    "rejected": "That was refused.",
    "no-session": "Could not start a session. Try again in a moment.",
    "failed": "It could not be sent. Try again.",
}

export function sendFailureOf(e: unknown): SendFailure {
    if (e instanceof SupportRateLimitedError) return "rate-limited"
    if (e instanceof SupportRejectedError) return "rejected"
    if (e instanceof SupportNoSessionError) return "no-session"
    return "failed"
}
