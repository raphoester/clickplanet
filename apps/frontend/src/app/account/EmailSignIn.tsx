import {FormEvent, useId, useState} from "react"
import {CODE_LENGTH, Intent} from "../../backends/account.ts"
import {AccountState, AccountStore, PendingCode} from "./accountStore.ts"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type EmailSignInProps = {
    state: Ready
    store: AccountStore
    intent: Intent
}

function looksLikeAnAddress(address: string): boolean {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(address.trim())
}

export default function EmailSignIn({state, store, intent}: EmailSignInProps) {
    return state.code
        ? <CodeForm state={state} store={store} code={state.code}/>
        : <AddressForm state={state} store={store} intent={intent}/>
}

function AddressForm({state, store, intent}: EmailSignInProps) {
    const [address, setAddress] = useState("")
    const inputId = useId()
    const canSend = state.busy === undefined && looksLikeAnAddress(address)
    const besideButtons = state.offered.some((p) => p !== "email" && !state.me.linked.includes(p))
    const label = intent === "link" ? "Link an email" : besideButtons ? "Or with your email" : "With your email"

    const submit = (event: FormEvent) => {
        event.preventDefault()
        if (canSend) void store.sendCode(address.trim(), intent)
    }

    return <form className="account-name account-email" onSubmit={submit} aria-busy={state.busy === "sendCode"}>
        <label className="menu-label" htmlFor={inputId}>{label}</label>
        <div className="account-name-row">
            <input id={inputId}
                   className="account-name-input"
                   type="email"
                   inputMode="email"
                   autoComplete="email"
                   autoCapitalize="off"
                   spellCheck={false}
                   placeholder="you@example.com"
                   value={address}
                   onChange={(e) => setAddress(e.target.value)}/>
            <button type="submit" className="button button-mini account-name-save" disabled={!canSend}>
                Send code
            </button>
        </div>
        <p className="account-name-hint">We email you a code. No password.</p>
    </form>
}

function CodeForm({state, store, code}: {state: Ready, store: AccountStore, code: PendingCode}) {
    const [draft, setDraft] = useState("")
    const inputId = useId()
    const digits = draft.replace(/\D/g, "")
    const busy = state.busy !== undefined
    const canCheck = !busy && digits.length === CODE_LENGTH

    const submit = (event: FormEvent) => {
        event.preventDefault()
        if (canCheck) void store.checkCode(digits)
    }

    return <form className="account-name account-email" onSubmit={submit} aria-busy={state.busy === "checkCode"}>
        <label className="menu-label" htmlFor={inputId}>Code</label>
        <p className="account-name-hint">
            We sent a code to <strong className="account-email-address">{code.address}</strong>. It works for 10 minutes.
        </p>
        <div className="account-name-row">
            <input id={inputId}
                   className="account-name-input account-code-input"
                   inputMode="numeric"
                   autoComplete="one-time-code"
                   maxLength={CODE_LENGTH * 2}
                   placeholder="123456"
                   value={draft}
                   onChange={(e) => setDraft(e.target.value)}/>
            <button type="submit" className="button button-mini account-name-save" disabled={!canCheck}>
                {code.intent === "link" ? "Link" : "Sign in"}
            </button>
        </div>
        <div className="account-email-actions">
            <button type="button" className="account-text-button" disabled={busy}
                    onClick={() => void store.sendCode(code.address, code.intent)}>
                Send a new code
            </button>
            <button type="button" className="account-text-button" disabled={busy} onClick={() => store.cancelCode()}>
                Use another address
            </button>
        </div>
    </form>
}
