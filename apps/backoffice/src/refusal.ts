import {Code, ConnectError} from "@connectrpc/connect"
import {clockOf, timeOf} from "./components/time.ts"

export type Refusal = "signedOut" | "notOperator" | "startAgain" | "failed"

export type Failure = {refusal: Refusal, detail: string}

export function failureOf(err: unknown): Failure {
    const error = ConnectError.from(err)
    return {refusal: refusalOf(error.code), detail: error.rawMessage}
}

function refusalOf(code: Code): Refusal {
    if (code === Code.Unauthenticated) return "signedOut"
    if (code === Code.PermissionDenied) return "notOperator"
    if (code === Code.FailedPrecondition) return "startAgain"
    return "failed"
}

const STAMP = /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z/g
const UTC_CLOCK = /\b(\d{2}):(\d{2}) UTC\b/g

// The server's words with each UTC time in the operator's own: a time of day on the day of the first stamp, or today.
export function localized(message: string, now: Date = new Date()): string {
    const day = new Date(message.match(STAMP)?.[0] ?? now)
    return message
        .replace(UTC_CLOCK, (_, hours: string, minutes: string) =>
            clockOf(new Date(Date.UTC(day.getUTCFullYear(), day.getUTCMonth(), day.getUTCDate(), Number(hours), Number(minutes)))))
        .replace(STAMP, stamp => timeOf(new Date(stamp)))
}

// What not_found, failed_precondition and unimplemented mean depend on the action refused, so each says its own.
export type Meanings = {notFound?: string, failedPrecondition?: string, unimplemented?: string}

// A refused action in the operator's words.
export function wordsOf(err: unknown, meanings: Meanings = {}): string {
    const error = ConnectError.from(err)

    switch (error.code) {
        case Code.NotFound:
            if (meanings.notFound !== undefined) return meanings.notFound
            break
        case Code.FailedPrecondition:
            if (meanings.failedPrecondition !== undefined) return meanings.failedPrecondition
            break
        case Code.Unimplemented:
            if (meanings.unimplemented !== undefined) return meanings.unimplemented
            break
        case Code.InvalidArgument:
            return `The server refused it: ${localized(error.rawMessage)}`
        case Code.Unauthenticated:
            return "Your session ended. Reload the page and sign in again."
        case Code.PermissionDenied:
            return "This account is not an operator."
        case Code.DeadlineExceeded:
            return "The server took too long to answer. It may have done part of it: look before you run it again."
    }
    return `The call failed: ${localized(error.rawMessage)}`
}
