import {CSSProperties, FormEvent, useId, useState} from "react"
import {PROVIDER_NAMES} from "../../backends/account.ts"
import {isValidUsername, MAX_USERNAME_LENGTH, MIN_USERNAME_LENGTH, NameColor, usernameOf} from "../../backends/player.ts"
import {AccountState, AccountStore} from "./accountStore.ts"
import {colorMessageOf, messageOf, providerList, usernameMessageOf} from "./authMessages.ts"
import {factor} from "../../domain/clickPrice.ts"
import {authorHue, NAME_COLORS} from "../../domain/authorColor.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import {UserIcon} from "../components/icons.tsx"
import ProviderButton from "./ProviderButton.tsx"
import EmailSignIn from "./EmailSignIn.tsx"
import ProgressTab from "./ProgressTab.tsx"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type AccountButtonProps = {
    state: Ready
    onOpen: () => void
    buttonRef?: React.Ref<HTMLButtonElement>
}

export function AccountButton({state, onOpen, buttonRef}: AccountButtonProps) {
    const label = state.me.linked.length > 0 ? "Account" : "Sign in"
    return <button ref={buttonRef}
                   type="button"
                   className="button menu-icon"
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
    linkedMultiplier?: number
}

type Tab = "progress" | "settings"

export default function AccountPanel(props: AccountPanelProps) {
    const [tab, setTab] = useState<Tab>("progress")
    const tabsId = useId()

    if (props.state.me.linked.length === 0) return <AccountSettings {...props}/>

    const tabButton = (value: Tab, label: string) => <button type="button"
                                                              role="tab"
                                                              id={`${tabsId}-${value}`}
                                                              aria-selected={tab === value}
                                                              aria-controls={`${tabsId}-panel`}
                                                              className={`button button-mini account-tab${tab === value ? " button-secondary" : ""}`}
                                                              onClick={() => setTab(value)}>
        {label}
    </button>

    return <div className="account-tabbed">
        <div className="account-tabs" role="tablist" aria-label="Account">
            {tabButton("progress", "Progress")}
            {tabButton("settings", "Settings")}
        </div>
        <div role="tabpanel" id={`${tabsId}-panel`} aria-labelledby={`${tabsId}-${tab}`}>
            {tab === "progress"
                ? <ProgressTab store={props.store} me={props.state.me}/>
                : <AccountSettings {...props}/>}
        </div>
    </div>
}

function AccountSettings({state, store, onDelete, linkedMultiplier}: AccountPanelProps) {
    const busy = state.busy !== undefined
    const linked = state.me.linked
    const toLink = state.offered.filter((p) => !linked.includes(p))
    const buttons = toLink.filter((p) => p !== "email")

    return <div className="account-panel" aria-busy={busy}>
        {linked.length === 0
            ? <p className="account-text">
                {linkedMultiplier
                    ? `Sign in to click ${factor(linkedMultiplier)}× faster and keep your stats on every device.`
                    : "Sign in to keep your stats on every device."} You do not need an account to play.
            </p>
            : <p className="account-text">Signed in with {providerList(linked)}.</p>}

        {linked.length > 0 && <UsernameForm key={state.username ?? ""} state={state} store={store}/>}
        {linked.length > 0 && state.username !== undefined && <ColorPicker name={state.username} state={state} store={store}/>}

        {buttons.map((provider) => <ProviderButton key={provider}
                                                   provider={provider}
                                                   label={`${linked.length === 0 ? "Sign in with" : "Link"} ${PROVIDER_NAMES[provider]}`}
                                                   disabled={busy}
                                                   onClick={() => void (linked.length === 0 ? store.signIn(provider) : store.link(provider))}/>)}

        {toLink.includes("email") && <EmailSignIn state={state} store={store} intent={linked.length === 0 ? "signIn" : "link"}/>}

        {linked.length > 0 && <>
            <button type="button"
                    className="button account-button"
                    disabled={busy}
                    onClick={() => void store.signOut()}>
                Sign out
            </button>
            <button type="button"
                    className="button account-button"
                    disabled={busy}
                    onClick={() => void store.signOutEverywhere()}>
                Sign out everywhere
            </button>
            <button type="button"
                    className="button account-button account-delete"
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

// maxLength counts UTF-16 units; the username rule counts code points.
const MAX_INPUT_LENGTH = MAX_USERNAME_LENGTH * 2

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
                   className="field account-name-input"
                   value={draft}
                   autoComplete="off"
                   autoCapitalize="off"
                   spellCheck={false}
                   maxLength={MAX_INPUT_LENGTH}
                   placeholder="Pick a username"
                   aria-describedby={hintId}
                   onChange={(e) => setDraft(e.target.value)}/>
            <button type="submit"
                    className="button button-mini button-secondary account-name-save"
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

function ColorPicker({name, state, store}: {name: string, state: Ready, store: AccountStore}) {
    const labelId = useId()
    const chosen = state.color ?? NameColor.UNSPECIFIED
    const blocked = state.busy !== undefined || state.naming === true || state.coloring === true

    const swatch = (color: NameColor, label: string, hue: number) => {
        const selected = color === chosen
        return <button key={color}
                       type="button"
                       className={color === NameColor.UNSPECIFIED ? "account-color-swatch account-color-swatch-auto" : "account-color-swatch"}
                       style={{"--author-hue": hue} as CSSProperties}
                       aria-label={label}
                       aria-pressed={selected}
                       title={label}
                       disabled={blocked}
                       onClick={() => {
                           if (!selected) void store.setColor(color)
                       }}/>
    }

    return <div className="account-color" aria-busy={state.coloring === true}>
        <div className="account-name-head">
            <span className="menu-label" id={labelId}>Name color</span>
            <span className="menu-label account-color-preview" style={authorStyle({name, color: chosen, guest: false})}>
                {name}
            </span>
        </div>
        <div className="account-color-swatches" role="group" aria-labelledby={labelId}>
            {swatch(NameColor.UNSPECIFIED, "From your name", authorHue(name))}
            {NAME_COLORS.map((choice) => swatch(choice.color, choice.label, choice.hue))}
        </div>
        <p className="account-name-hint">Everyone sees it in the chat and the player list. Guests are grey.</p>
        {state.colorFailure && <p className="account-failure" role="alert">{colorMessageOf(state.colorFailure)}</p>}
    </div>
}
