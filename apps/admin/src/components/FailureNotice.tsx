import {Failure} from "../refusal.ts"

const GAME_URL = "https://clickplanet.lol/play"

export function FailureNotice({failure, onRetry}: {failure: Failure, onRetry: () => void}) {
    return (
        <div className="notice" role="alert">
            <p className="notice-title">{titleOf(failure)}</p>
            {failure.detail !== "" && <p className="muted mono">{failure.detail}</p>}
            <button type="button" className="button" onClick={onRetry}>Try again</button>
        </div>
    )
}

function titleOf(failure: Failure) {
    switch (failure.refusal) {
        case "signedOut":
            return <>Sign in on <a href={GAME_URL}>clickplanet.lol</a>, then try again.</>
        case "notOperator":
            return <>This account is not an operator.</>
        case "failed":
            return <>The API did not answer.</>
    }
}
