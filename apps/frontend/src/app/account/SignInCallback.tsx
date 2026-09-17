import {useEffect, useRef, useState} from "react"
import {AuthFailure, failureOf} from "../../backends/account.ts"
import {SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {messageOf, retryOf} from "./authMessages.ts"
import "./Account.css"

export type SignInCallbackProps = {
    callback: Callback
    /** Trades the code, and invalidates the click token on success. */
    complete: (code: string, state: string) => Promise<void>
    /** Goes back to the provider the sign-in started with. Absent when none is remembered. */
    startAgain?: () => Promise<void>
    /** Back to the game. */
    onDone: () => void
}

type Step =
    | {kind: "working"}
    | {kind: "failed", failure: AuthFailure}

/**
 * The page a provider sends the browser back to. It trades the code once, then
 * hands over to the game. It never shows the code: `main.tsx` has already taken
 * it out of the address bar.
 */
export default function SignInCallback(props: SignInCallbackProps) {
    const [step, setStep] = useState<Step>({kind: "working"})
    // StrictMode runs an effect twice, and a code is good once: the second
    // trade would fail and replace the first one's success.
    const started = useRef(false)

    const complete = async () => {
        if (props.callback.kind !== "code") return
        setStep({kind: "working"})
        try {
            await props.complete(props.callback.code, props.callback.state)
            props.onDone()
        } catch (e) {
            setStep({kind: "failed", failure: failureOf(e)})
        }
    }

    const startAgain = async () => {
        if (!props.startAgain) return
        setStep({kind: "working"})
        try {
            await props.startAgain()
        } catch (e) {
            setStep({kind: "failed", failure: failureOf(e)})
        }
    }

    useEffect(() => {
        if (started.current) return
        started.current = true
        void complete()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const back = <button type="button" className="button button-ghost" onClick={props.onDone}>
        Back to the game
    </button>

    if (props.callback.kind === "declined") {
        return <Card title="You did not sign in">
            <p>You can play without an account.</p>
            <div className="sign-in-callback-actions">
                {props.startAgain && <button type="button" className="button sign-in-callback-primary" onClick={() => void startAgain()}>
                    Try again
                </button>}
                {back}
            </div>
        </Card>
    }

    if (props.callback.kind === "invalid") {
        return <Card title="This sign-in link is not complete">
            <p>Start the sign-in again from the menu.</p>
            <div className="sign-in-callback-actions">{back}</div>
        </Card>
    }

    if (step.kind === "working") {
        return <Card title="Signing you in…" working/>
    }

    const retry = retryOf(step.failure)
    return <Card title="Sign-in did not work">
        <p role="alert">{messageOf(step.failure)}</p>
        <div className="sign-in-callback-actions">
            {retry === "complete" && <button type="button" className="button sign-in-callback-primary" onClick={() => void complete()}>
                Try again
            </button>}
            {retry === "start" && props.startAgain && <button type="button" className="button sign-in-callback-primary" onClick={() => void startAgain()}>
                Try again
            </button>}
            {back}
        </div>
    </Card>
}

function Card(props: {title: string, working?: boolean, children?: React.ReactNode}) {
    return <main className="sign-in-callback">
        <div className="sign-in-callback-card" aria-live="polite">
            {props.working && <div className="sign-in-callback-spinner"/>}
            <h1>{props.title}</h1>
            {props.children}
        </div>
    </main>
}
