import {Code, ConnectError} from "@connectrpc/connect"

export type Refusal = "signedOut" | "notOperator" | "failed"

export type Failure = {refusal: Refusal, detail: string}

export function failureOf(err: unknown): Failure {
    const error = ConnectError.from(err)
    return {refusal: refusalOf(error.code), detail: error.rawMessage}
}

function refusalOf(code: Code): Refusal {
    if (code === Code.Unauthenticated) return "signedOut"
    if (code === Code.PermissionDenied) return "notOperator"
    return "failed"
}
