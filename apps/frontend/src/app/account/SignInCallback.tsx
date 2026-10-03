import {useEffect, useRef, useState} from "react"
import {AuthFailure, failureOf, Provider} from "../../backends/account.ts"
import {SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {messageOf, retryOf} from "./authMessages.ts"
import "./Account.css"

export type SignInCallbackProps = {
    callback: Callback
    provider?: Provider
    complete: (code: string, state: string) => Promise<void>
    startAgain?: () => Promise<void>
    onDone: () => void
}

type Step =
    | {kind: "working"}
    | {kind: "failed", failure: AuthFailure}

export default function SignInCallback(props: SignInCallbackProps) {
    const [step, setStep] = useState<Step>({kind: "working"})
    // StrictMode runs effects twice, and a sign-in code can be traded only once.
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

    const back = <button type="button" className="button" onClick={props.onDone}>
        Back to the game
    </button>

    if (props.callback.kind === "declined") {
        return <Card title="You did not sign in">
            <p>You can play without an account.</p>
            <div className="sign-in-callback-actions">
                {props.startAgain && <button type="button" className="button button-action" onClick={() => void startAgain()}>
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
    const notLinked = step.failure === "linkedElsewhere" || step.failure === "alreadyLinked"
    return <Card title={notLinked ? "Not linked" : "Sign-in did not work"}>
        <p role="alert">{messageOf(step.failure, props.provider)}</p>
        <div className="sign-in-callback-actions">
            {retry === "complete" && <button type="button" className="button button-action" onClick={() => void complete()}>
                Try again
            </button>}
            {retry === "start" && props.startAgain && <button type="button" className="button button-action" onClick={() => void startAgain()}>
                Try again
            </button>}
            {back}
        </div>
    </Card>
}

function Card(props: {title: string, working?: boolean, children?: React.ReactNode}) {
    return <main className="sign-in-callback">
        <div className="panel sign-in-callback-card" aria-live="polite">
            {props.working && <div className="sign-in-callback-spinner"/>}
            <h1>{props.title}</h1>
            {props.children}
        </div>
    </main>
}
