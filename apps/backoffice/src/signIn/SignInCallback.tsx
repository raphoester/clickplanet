import {useEffect, useRef, useState} from "react"
import {authAdmin} from "../api.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Failure, failureOf} from "../refusal.ts"
import {Callback} from "./callback.ts"
import {SignInButton} from "./SignInButton.tsx"

export function SignInCallback({callback, onDone}: {callback: Callback, onDone: () => void}) {
    const [failure, setFailure] = useState<Failure | undefined>(undefined)
    // A code is good once, and StrictMode runs an effect twice.
    const sent = useRef(false)

    useEffect(() => {
        if (callback.kind !== "code" || sent.current) return
        sent.current = true
        authAdmin.completeSignIn({code: callback.code, state: callback.state}).then(
            () => onDone(),
            (err: unknown) => setFailure(failureOf(err)),
        )
    }, [callback, onDone])

    switch (callback.kind) {
        case "refused":
            return <Refused title="Google did not sign you in."/>
        case "broken":
            return <Refused title="This sign-in link is broken."/>
        case "code":
            if (failure !== undefined) return <FailureNotice failure={failure} onRetry={onDone}/>
            return <p className="muted">Signing in…</p>
    }
}

function Refused({title}: {title: string}) {
    return (
        <div className="notice" role="alert">
            <p className="notice-title">{title}</p>
            <SignInButton/>
        </div>
    )
}
