import {useState} from "react"
import {authAdmin} from "../api.ts"
import {Failure, failureOf} from "../refusal.ts"

export function SignInButton() {
    const [going, setGoing] = useState(false)
    const [failure, setFailure] = useState<Failure | undefined>(undefined)

    const start = () => {
        setGoing(true)
        setFailure(undefined)
        authAdmin.startSignIn({}).then(
            answer => window.location.assign(answer.authorizationUrl),
            (err: unknown) => {
                setGoing(false)
                setFailure(failureOf(err))
            },
        )
    }

    return (
        <div className="sign-in">
            <button type="button" className="button button-primary" disabled={going} onClick={start}>
                Sign in with Google
            </button>
            {failure !== undefined && <p className="muted mono">{failure.detail}</p>}
        </div>
    )
}
