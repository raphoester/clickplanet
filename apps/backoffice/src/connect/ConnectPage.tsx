import {PromiseClient} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {AdminService as AuthAdminService} from "../gen/grpc/auth/v1/admin_connect.ts"
import {Failure, failureOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"

export type ConnectAdmin = Pick<PromiseClient<typeof AuthAdminService>, "describeConnection" | "approveConnection">

type Props = {
    request: string
    auth: ConnectAdmin
    // Where the browser goes once the operator said yes or no: back to the app.
    leave: (url: string) => void
    onDismiss: () => void
}

// An app such as Claude asks to operate as the signed-in operator: the operator says yes or no, here and nowhere
// else. Yes sends the browser back to the app with a code; no, with a refusal.
export function ConnectPage({request, auth, leave, onDismiss}: Props) {
    const describe = useCallback((signal: AbortSignal) => auth.describeConnection({request}, {signal}), [auth, request])
    const [described, again] = useCall(describe)
    const [sending, setSending] = useState(false)
    const [failure, setFailure] = useState<string | undefined>(undefined)

    if (described.state === "loading") return <p className="muted">Reading the request…</p>
    if (described.state === "failed") {
        return <Dismissed words={wordsOf(described)} onRetry={again} onDismiss={onDismiss}/>
    }

    const {clientName, redirectHost, denyUrl} = described.value
    const allow = () => {
        setSending(true)
        setFailure(undefined)
        auth.approveConnection({request}).then(
            answer => leave(answer.redirectUrl),
            (err: unknown) => {
                setSending(false)
                setFailure(wordsOf(failureOf(err)))
            },
        )
    }

    return (
        <div className="panel connect">
            <h2 className="page-title">Connect {clientName}</h2>
            <p>
                <strong>{clientName}</strong> asks to use every operator tool as you: read the players, the reports
                and the notes, and ban, mute, revert and repaint on the live game. It will be sent back
                to <code className="mono">{redirectHost}</code>.
            </p>
            <p className="muted">
                Only allow it if you just added the connector yourself. It holds the access until you take your
                account's operator flag away.
            </p>
            {failure !== undefined && <p className="notice" role="alert">{failure}</p>}
            <div className="actions">
                <button type="button" className="button button-primary" disabled={sending} onClick={allow}>
                    Allow
                </button>
                <button type="button" className="button" disabled={sending} onClick={() => leave(denyUrl)}>
                    Deny
                </button>
            </div>
        </div>
    )
}

function Dismissed({words, onRetry, onDismiss}: {words: string, onRetry: () => void, onDismiss: () => void}) {
    return (
        <div className="notice" role="alert">
            <p className="notice-title">{words}</p>
            <div className="actions">
                <button type="button" className="button" onClick={onRetry}>Try again</button>
                <button type="button" className="button" onClick={onDismiss}>Go to the backoffice</button>
            </div>
        </div>
    )
}

// The page already knows the operator is signed in: what is left is a request too old, or one nobody sealed.
function wordsOf(failure: Failure): string {
    switch (failure.refusal) {
        case "startAgain":
            return "This request expired. Connect again from the app."
        case "signedOut":
            return "Your session ended. Reload the page and sign in again."
        case "notOperator":
            return "This account is not an operator."
        case "failed":
            return `ClickPlanet could not read this request: ${failure.detail}. Connect again from the app.`
    }
}
