import {FormEvent, useId, useState} from "react"
import {PROVIDER_NAMES} from "../../backends/account.ts"
import {isValidUsername, MAX_USERNAME_LENGTH, MIN_USERNAME_LENGTH, usernameOf} from "../../backends/player.ts"
import {AccountState, AccountStore} from "./accountStore.ts"
import {messageOf, providerList, usernameMessageOf} from "./authMessages.ts"
import {factor} from "../../domain/clickPrice.ts"
import {UserIcon} from "../components/icons.tsx"
import ProviderButton from "./ProviderButton.tsx"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type AccountButtonProps = {
    state: Ready
    onOpen: () => void
    buttonRef?: React.Ref<HTMLButtonElement>
}

/** One icon in the menu's actions. Login is optional, so it asks for nothing more than this. */
export function AccountButton({state, onOpen, buttonRef}: AccountButtonProps) {
    const label = state.me.linked.length > 0 ? "Account" : "Sign in"
    return <button ref={buttonRef}
                   type="button"
                   className="button button-ghost menu-icon"
                   aria-label={label}
                   title={label}
                   onClick={onOpen}>
        <UserIcon size={26}/>
    </button>
}

export type AccountPanelProps = {
    state: Ready
    store: AccountStore
    onDelete: () => void
    /** What signing in multiplies the click allowance by, as the server said. */
    linkedMultiplier?: number
}

export default function AccountPanel({state, store, onDelete, linkedMultiplier}: AccountPanelProps) {
    const busy = state.busy !== undefined
    const linked = state.me.linked
    const toLink = state.offered.filter((p) => !linked.includes(p))

    return <div className="account-panel" aria-busy={busy}>
        {linked.length === 0
            ? <p className="account-text">
                {linkedMultiplier
                    ? `Sign in to click ${factor(linkedMultiplier)}× faster and keep your stats on every device.`
                    : "Sign in to keep your stats on every device."} You do not need an account to play.
            </p>
            : <p className="account-text">Signed in with {providerList(linked)}.</p>}

        {/* Keyed on the name, so a read or a save that lands resets what is typed. */}
        {linked.length > 0 && <UsernameForm key={state.username ?? ""} state={state} store={store}/>}

        {toLink.map((provider) => <ProviderButton key={provider}
                                                  provider={provider}
                                                  label={`${linked.length === 0 ? "Sign in with" : "Link"} ${PROVIDER_NAMES[provider]}`}
                                                  disabled={busy}
                                                  onClick={() => void (linked.length === 0 ? store.signIn(provider) : store.link(provider))}/>)}

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

/** The input counts UTF-16 units, the rule code points: room for letters past the BMP and spaces at the ends. */
const MAX_INPUT_LENGTH = MAX_USERNAME_LENGTH * 2

/** The username the chat shows. Only a linked account has one. */
function UsernameForm({state, store}: {state: Ready, store: AccountStore}) {
    const current = state.username ?? ""
    const [draft, setDraft] = useState(current)
    const inputId = useId()
    const hintId = useId()

    const name = usernameOf(draft)
    const blocked = state.busy !== undefined || state.naming === true
    const canSave = !blocked && name !== current && isValidUsername(name)

    const submit = (event: FormEvent) => {
        event.preventDefault()
        if (canSave) void store.setUsername(name)
    }

    return <form className="account-name" onSubmit={submit} aria-busy={state.naming === true}>
        <div className="account-name-head">
            <label className="menu-label" htmlFor={inputId}>Username</label>
            <span className="menu-label account-name-current">{current || "none yet"}</span>
        </div>
        <div className="account-name-row">
            <input id={inputId}
                   className="account-name-input"
                   value={draft}
                   autoComplete="off"
                   autoCapitalize="off"
                   spellCheck={false}
                   maxLength={MAX_INPUT_LENGTH}
                   placeholder="Pick a username"
                   aria-describedby={hintId}
                   onChange={(e) => setDraft(e.target.value)}/>
            <button type="submit"
                    className="button button-mini account-name-save"
                    disabled={!canSave}>
                Save
            </button>
        </div>
        <p className="account-name-hint" id={hintId}>
            {MIN_USERNAME_LENGTH}–{MAX_USERNAME_LENGTH} letters, digits, spaces or _. Shown in the chat.
        </p>
        {state.nameFailure && <p className="account-failure" role="alert">{usernameMessageOf(state.nameFailure)}</p>}
    </form>
}
