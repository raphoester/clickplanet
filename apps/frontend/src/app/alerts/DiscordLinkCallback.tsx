import {useEffect, useRef, useState} from "react"
import {AlertsFailure, alertsFailureOf} from "../../backends/alerts.ts"
import {SignInCallback as Callback} from "../../domain/signInCallback.ts"
import {CallbackCard} from "../account/SignInCallback.tsx"
import {DiscordIcon} from "../components/icons.tsx"
import {linkMessageOf} from "./alertsMessages.ts"

export type DiscordLinkCallbackProps = {
    callback: Callback
    complete: (code: string, state: string) => Promise<string>
    startAgain: () => Promise<void>
    onDone: () => void
}

type Step =
    | {kind: "waiting"}
    | {kind: "trading"}
    | {kind: "leaving"}
    | {kind: "linked", serverUrl: string}
    | {kind: "failed", failure: AlertsFailure}

export default function DiscordLinkCallback(props: DiscordLinkCallbackProps) {
    const [step, setStep] = useState<Step>(props.callback.kind === "code" ? {kind: "trading"} : {kind: "waiting"})
    // StrictMode runs effects twice, and a Discord code can be traded only once.
    const started = useRef(false)

    useEffect(() => {
        if (started.current || props.callback.kind !== "code") return
        started.current = true
        props.complete(props.callback.code, props.callback.state).then(
            (serverUrl) => setStep({kind: "linked", serverUrl}),
            (e) => setStep({kind: "failed", failure: alertsFailureOf(e)}),
        )
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const startAgain = async () => {
        setStep({kind: "leaving"})
        try {
            await props.startAgain()
        } catch (e) {
            setStep({kind: "failed", failure: alertsFailureOf(e)})
        }
    }

    const tryAgain = <button type="button" className="button button-action" onClick={() => void startAgain()}>
        Try again
    </button>

    const back = <button type="button" className="button" onClick={props.onDone}>
        Back to the game
    </button>

    switch (step.kind) {
        case "trading":
            return <CallbackCard title="Linking Discord…" working/>
        case "leaving":
            return <CallbackCard title="Opening Discord…" working/>
        case "linked":
            return <CallbackCard title="You are in!">
                <div className="sign-in-callback-actions">
                    {step.serverUrl && <a href={step.serverUrl}
                                          target="_blank"
                                          rel="noopener noreferrer"
                                          className="button button-action">
                        <DiscordIcon size={20}/>
                        Open Discord
                    </a>}
                    {back}
                </div>
            </CallbackCard>
        case "failed":
            return <CallbackCard title="Not linked">
                <p role="alert">{linkMessageOf(step.failure)}</p>
                <div className="sign-in-callback-actions">
                    {step.failure !== "off" && tryAgain}
                    {back}
                </div>
            </CallbackCard>
        case "waiting":
            return <CallbackCard title={props.callback.kind === "declined" ? "You did not link Discord" : "This link is not complete"}>
                <div className="sign-in-callback-actions">
                    {tryAgain}
                    {back}
                </div>
            </CallbackCard>
    }
}
