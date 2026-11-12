import {useEffect, useRef, useState} from "react"
import {AuthFailure, failureOf, Provider} from "../../backends/account.ts"
import {SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {messageOf, retryOf} from "./authMessages.ts"
import {t} from "../../i18n/page.ts"
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
        {t("signIn.backToGame")}
    </button>

    if (props.callback.kind === "declined") {
        return <CallbackCard title={t("signIn.declined")}>
            <p>{t("signIn.playWithout")}</p>
            <div className="sign-in-callback-actions">
                {props.startAgain && <button type="button" className="button button-action" onClick={() => void startAgain()}>
                    {t("common.retry")}
                </button>}
                {back}
            </div>
        </CallbackCard>
    }

    if (props.callback.kind === "invalid") {
        return <CallbackCard title={t("signIn.linkIncomplete")}>
            <p>{t("signIn.startFromMenu")}</p>
            <div className="sign-in-callback-actions">{back}</div>
        </CallbackCard>
    }

    if (step.kind === "working") {
        return <CallbackCard title={t("signIn.working")} working/>
    }

    const retry = retryOf(step.failure)
    const notLinked = step.failure === "linkedElsewhere" || step.failure === "alreadyLinked"
    return <CallbackCard title={t(notLinked ? "signIn.notLinked" : "signIn.failed")}>
        <p role="alert">{messageOf(step.failure, props.provider)}</p>
        <div className="sign-in-callback-actions">
            {retry === "complete" && <button type="button" className="button button-action" onClick={() => void complete()}>
                {t("common.retry")}
            </button>}
            {retry === "start" && props.startAgain && <button type="button" className="button button-action" onClick={() => void startAgain()}>
                {t("common.retry")}
            </button>}
            {back}
        </div>
    </CallbackCard>
}

export function CallbackCard(props: {title: string, working?: boolean, children?: React.ReactNode}) {
    return <main className="sign-in-callback">
        <div className="panel sign-in-callback-card" aria-live="polite">
            {props.working && <div className="sign-in-callback-spinner"/>}
            <h1>{props.title}</h1>
            {props.children}
        </div>
    </main>
}
