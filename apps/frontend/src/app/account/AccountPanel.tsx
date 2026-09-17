import {PROVIDER_NAMES} from "../../backends/account.ts"
import {AccountState, AccountStore} from "./accountStore.ts"
import {messageOf, providerList} from "./authMessages.ts"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type AccountRowProps = {
    state: Ready
    onOpen: () => void
    buttonRef?: React.Ref<HTMLButtonElement>
}

/** One line in the menu. Login is optional, so it asks for nothing more than this. */
export function AccountRow({state, onOpen, buttonRef}: AccountRowProps) {
    const linked = state.me.linked.length > 0
    return <div className="account-row">
        <span className="account-row-text">
            {linked ? `Signed in with ${providerList(state.me.linked)}` : "Keep your stats on every device"}
        </span>
        <button ref={buttonRef}
                type="button"
                className="button button-mini account-row-button"
                onClick={onOpen}>
            {linked ? "Account" : "Sign in"}
        </button>
    </div>
}

export type AccountPanelProps = {
    state: Ready
    store: AccountStore
    onDelete: () => void
}

export default function AccountPanel({state, store, onDelete}: AccountPanelProps) {
    const busy = state.busy !== undefined
    const linked = state.me.linked
    const toLink = state.offered.filter((p) => !linked.includes(p))

    return <div className="account-panel" aria-busy={busy}>
        {linked.length === 0
            ? <p className="account-text">
                Sign in to keep your stats on every device. You do not need an account to play.
            </p>
            : <p className="account-text">Signed in with {providerList(linked)}.</p>}

        {toLink.map((provider) => <button key={provider}
                                          type="button"
                                          className="button button-ghost account-button"
                                          disabled={busy}
                                          onClick={() => void store.signIn(provider)}>
            {linked.length === 0 ? "Sign in with" : "Link"} {PROVIDER_NAMES[provider]}
        </button>)}

        {linked.length > 0 && <>
            <button type="button"
                    className="button button-ghost account-button"
                    disabled={busy}
                    onClick={() => void store.signOut()}>
                Sign out
            </button>
            <button type="button"
                    className="button button-ghost account-button"
                    disabled={busy}
                    onClick={() => void store.signOutEverywhere()}>
                Sign out everywhere
            </button>
            <button type="button"
                    className="button button-ghost account-button account-delete"
                    disabled={busy}
                    onClick={onDelete}>
                Delete account
            </button>
        </>}

        {state.failure && <p className="account-failure" role="alert">{messageOf(state.failure)}</p>}

        <p className="account-legal">
            <a href="/privacy" target="_blank" rel="noopener noreferrer">Privacy policy</a>
        </p>
    </div>
}
