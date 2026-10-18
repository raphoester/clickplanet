import {Failure} from "../refusal.ts"
import {SignInButton} from "../signIn/SignInButton.tsx"

export function FailureNotice({failure, onRetry}: {failure: Failure, onRetry: () => void}) {
    return (
        <div className="notice" role="alert">
            <p className="notice-title">{titleOf(failure)}</p>
            {failure.detail !== "" && <p className="muted mono">{failure.detail}</p>}
            {failure.refusal === "failed"
                ? <button type="button" className="button" onClick={onRetry}>Try again</button>
                : <SignInButton/>}
        </div>
    )
}

function titleOf(failure: Failure): string {
    switch (failure.refusal) {
        case "signedOut":
            return "Sign in to continue."
        case "notOperator":
            return "This Google account is not an operator."
        case "startAgain":
            return "The sign-in expired. Start again."
        case "failed":
            return "The call failed."
    }
}
