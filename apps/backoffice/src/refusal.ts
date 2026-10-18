import {Code, ConnectError} from "@connectrpc/connect"

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

// What not_found and failed_precondition mean depend on the action refused, so each says its own.
export type Meanings = {notFound?: string, failedPrecondition?: string}

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
        case Code.InvalidArgument:
            return `The server refused it: ${error.rawMessage}`
        case Code.Unauthenticated:
            return "Your session ended. Reload the page and sign in again."
        case Code.PermissionDenied:
            return "This account is not an operator."
        case Code.DeadlineExceeded:
            return "The server took too long to answer. It may have done part of it: look before you run it again."
    }
    return `The call failed: ${error.rawMessage}`
}
