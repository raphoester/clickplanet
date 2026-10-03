import {FormEvent, useId, useState} from "react"
import {MarketingFailure} from "../../backends/marketing.ts"
import {SEASON_EMAILS_CONSENT} from "../../domain/seasonEmailsConsent.ts"
import {SeasonEmailsState, SeasonEmailsStore} from "./seasonEmailsStore.ts"
import "./SeasonEmails.css"

type Ready = Extract<SeasonEmailsState, {kind: "ready"}>

export type SeasonEmailsProps = {
    state: SeasonEmailsState
    store: SeasonEmailsStore
    onSignIn?: () => void
}

export default function SeasonEmails({state, store, onSignIn}: SeasonEmailsProps) {
    if (state.kind === "hidden") return null

    if (state.kind === "guest") {
        if (!onSignIn) return null
        return <div className="season-emails panel-box">
            <button type="button" className="button button-mini button-action season-emails-button" onClick={onSignIn}>
                {SEASON_EMAILS_CONSENT.words}
            </button>
        </div>
    }

    if (state.state === "none") return <OptIn key={state.address} state={state} store={store}/>

    const waiting = state.state === "waiting"
    return <div className="season-emails panel-box" aria-busy={state.busy === true}>
        <div className="season-emails-row">
            <p className="season-emails-status">
                {waiting ? `Confirm in the email sent to ${state.address}` : `Emails on · ${state.address}`}
            </p>
            <button type="button"
                    className="button button-mini season-emails-off"
                    disabled={state.busy === true}
                    onClick={() => void store.optOut()}>
                {waiting ? "Cancel" : "Turn off"}
            </button>
        </div>
        <Failure failure={state.failure}/>
    </div>
}

function OptIn({state, store}: {state: Ready, store: SeasonEmailsStore}) {
    const [draft, setDraft] = useState(state.address)
    const inputId = useId()
    const canSend = state.busy !== true && draft.trim() !== ""

    const submit = (event: FormEvent) => {
        event.preventDefault()
        if (canSend) void store.optIn(draft)
    }

    return <form className="season-emails panel-box" onSubmit={submit} noValidate aria-busy={state.busy === true}>
        <label className="menu-label" htmlFor={inputId}>Email</label>
        <input id={inputId}
               className="field season-emails-input"
               type="email"
               inputMode="email"
               autoComplete="email"
               autoCapitalize="off"
               spellCheck={false}
               value={draft}
               placeholder="you@example.com"
               onChange={(e) => setDraft(e.target.value)}/>
        <button type="submit" className="button button-mini button-action season-emails-button" disabled={!canSend}>
            {SEASON_EMAILS_CONSENT.words}
        </button>
        <Failure failure={state.failure}/>
    </form>
}

const MESSAGES: Record<MarketingFailure, string> = {
    invalid: "That is not an email address.",
    unavailable: "The mailing list did not answer. Try again later.",
    tooManyTries: "Too many tries. Wait a minute.",
    alreadySubscribed: "Emails are already on for another address.",
    guest: "Sign in first.",
    notSignedIn: "Sign in first.",
    off: "Season emails are off.",
    failed: "That did not work. Try again.",
}

function Failure({failure}: {failure?: MarketingFailure}) {
    if (!failure) return null
    return <p className="season-emails-failure" role="alert">{MESSAGES[failure]}</p>
}
