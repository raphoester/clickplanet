import {CSSProperties, FormEvent, useId, useState} from "react"
import {
    isValidUsername,
    MAX_USERNAME_LENGTH,
    MIN_USERNAME_LENGTH,
    NameColor,
    PlayerInfoBackend,
    usernameOf,
} from "../../backends/player.ts"
import {AccountState, AccountStore} from "./accountStore.ts"
import {colorMessageOf, messageOf, providerList, providerName, usernameMessageOf} from "./authMessages.ts"
import {factor} from "../../domain/clickPrice.ts"
import {NAME_COLORS, PickedColor} from "../../domain/authorColor.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import ProviderButton from "./ProviderButton.tsx"
import EmailSignIn from "./EmailSignIn.tsx"
import ProgressTab from "./ProgressTab.tsx"
import ClipsSwitch from "../clips/ClipsSwitch.tsx"
import {ClipsNameBackend} from "../../backends/clips.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type AccountPanelProps = {
    state: Ready
    store: AccountStore
    onDelete: () => void
    linkedMultiplier?: number
    playerInfo: PlayerInfoBackend
    listenForClicks?: ListenForClicks
    clips: ClipsNameBackend
}

type Tab = "progress" | "settings"

export default function AccountPanel(props: AccountPanelProps) {
    const [tab, setTab] = useState<Tab>("progress")
    const tabsId = useId()
    const words = useWordFilter()

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

    const name = props.state.username
    return <div className="account-tabbed">
        {name && <p className="account-who" style={authorStyle({color: props.state.color, guest: false})}>
            {words.masked(name)}
        </p>}
        <div className="account-tabs" role="tablist" aria-label={t("account.title")}>
            {tabButton("progress", t("account.progressTab"))}
            {tabButton("settings", t("account.title"))}
        </div>
        <div role="tabpanel" id={`${tabsId}-panel`} aria-labelledby={`${tabsId}-${tab}`}>
            {tab === "progress"
                ? <ProgressTab store={props.store}
                               me={props.state.me}
                               stats={name && props.state.accountId
                                   ? {backend: props.playerInfo, accountId: props.state.accountId}
                                   : undefined}
                               listenForClicks={props.listenForClicks}/>
                : <AccountSettings {...props}/>}
        </div>
    </div>
}

export function AccountSettings({state, store, onDelete, linkedMultiplier, clips}: Omit<AccountPanelProps, "playerInfo">) {
    const busy = state.busy !== undefined
    const linked = state.me.linked
    const toLink = state.offered.filter((p) => !linked.includes(p))
    const buttons = toLink.filter((p) => p !== "email")
    // CrazyGames signs the player in and out itself.
    const leavable = linked.length > 0 && !linked.includes("crazygames")

    return <div className="account-panel" aria-busy={busy}>
        {linked.length === 0
            ? <p className="account-text">
                {linkedMultiplier
                    ? t("account.signInFaster", {times: factor(linkedMultiplier)})
                    : t("account.signInStats")}
            </p>
            : <p className="account-text">{t("account.signedInWith", {providers: providerList(linked)})}</p>}

        {linked.length > 0 && <UsernameForm key={state.username ?? ""} state={state} store={store}/>}
        {linked.length > 0 && state.username !== undefined && <ColorPicker name={state.username} state={state} store={store}/>}
        {linked.length > 0 && state.username !== undefined && <ClipsSwitch key={state.accountId} clips={clips}/>}

        {buttons.map((provider) => <ProviderButton key={provider}
                                                   provider={provider}
                                                   label={t(linked.length === 0 ? "account.signInWith" : "account.link", {provider: providerName(provider)})}
                                                   disabled={busy}
                                                   onClick={() => void (linked.length > 0 && provider !== "crazygames" ? store.link(provider) : store.signIn(provider))}/>)}

        {toLink.includes("email") && <EmailSignIn state={state} store={store} intent={linked.length === 0 ? "signIn" : "link"}/>}

        {leavable && <>
            <button type="button"
                    className="button account-button"
                    disabled={busy}
                    onClick={() => void store.signOut()}>
                {t("account.signOut")}
            </button>
            <button type="button"
                    className="button account-button"
                    disabled={busy}
                    onClick={() => void store.signOutEverywhere()}>
                {t("account.signOutEverywhere")}
            </button>
        </>}
        {linked.length > 0 && <>
            <button type="button"
                    className="button account-button account-delete"
                    disabled={busy}
                    onClick={onDelete}>
                {t("account.delete")}
            </button>
        </>}

        {state.failure && <p className="account-failure" role="alert">{messageOf(state.failure)}</p>}

        <p className="account-legal">
            <a href="/privacy" target="_blank" rel="noopener noreferrer">{t("account.privacyPolicy")}</a>
        </p>
    </div>
}

// maxLength counts UTF-16 units; the username rule counts code points.
const MAX_INPUT_LENGTH = MAX_USERNAME_LENGTH * 2

const RENAME_DATE: Intl.DateTimeFormatOptions = {month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"}

export function UsernameForm({state, store}: {state: Ready, store: AccountStore}) {
    const current = state.username ?? ""
    const [draft, setDraft] = useState(current)
    const inputId = useId()
    const hintId = useId()

    const name = usernameOf(draft)
    const waiting = state.renamableAt !== undefined && state.renamableAt > Date.now() ? state.renamableAt : undefined
    const blocked = state.busy !== undefined || state.naming === true || waiting !== undefined
    const canSave = !blocked && name !== current && isValidUsername(name)
    const failure = state.nameFailure === "tooSoon" && waiting !== undefined ? undefined : state.nameFailure

    const submit = (event: FormEvent) => {
        event.preventDefault()
        if (canSave) void store.setUsername(name)
    }

    return <form className="account-name" onSubmit={submit} aria-busy={state.naming === true}>
        <div className="account-name-head">
            <label className="menu-label" htmlFor={inputId}>{t("username.label")}</label>
            <span className="menu-label account-name-current">{current || t("username.noneYet")}</span>
        </div>
        <div className="account-name-row">
            <input id={inputId}
                   className="field account-name-input"
                   value={draft}
                   autoComplete="off"
                   autoCapitalize="off"
                   spellCheck={false}
                   maxLength={MAX_INPUT_LENGTH}
                   placeholder={t("username.placeholder")}
                   aria-describedby={hintId}
                   disabled={waiting !== undefined}
                   onChange={(e) => setDraft(e.target.value)}/>
            <button type="submit"
                    className="button button-mini button-secondary account-name-save"
                    disabled={!canSave}>
                {t("common.save")}
            </button>
        </div>
        <p className="account-name-hint" id={hintId}>
            {waiting === undefined
                ? t("username.rule", {min: MIN_USERNAME_LENGTH, max: MAX_USERNAME_LENGTH})
                : t("username.changeOn", {date: translator().dateTime(waiting, RENAME_DATE)})}
        </p>
        {failure && <p className="account-failure" role="alert">{usernameMessageOf(failure)}</p>}
    </form>
}

const COLOR_NAMES: Record<PickedColor, Key> = {
    [NameColor.RED]: "color.red",
    [NameColor.ORANGE]: "color.orange",
    [NameColor.YELLOW]: "color.yellow",
    [NameColor.LIME]: "color.lime",
    [NameColor.GREEN]: "color.green",
    [NameColor.TEAL]: "color.teal",
    [NameColor.CYAN]: "color.cyan",
    [NameColor.BLUE]: "color.blue",
    [NameColor.INDIGO]: "color.indigo",
    [NameColor.VIOLET]: "color.violet",
    [NameColor.MAGENTA]: "color.magenta",
    [NameColor.PINK]: "color.pink",
}

export function ColorPicker({name, state, store}: {name: string, state: Ready, store: AccountStore}) {
    const labelId = useId()
    const words = useWordFilter()
    const chosen = state.color
    const blocked = state.busy !== undefined || state.naming === true

    const swatch = (color: PickedColor, hue: number) => {
        const selected = color === chosen
        const label = t(COLOR_NAMES[color])
        return <button key={color}
                       type="button"
                       className="account-color-swatch"
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
            <span className="menu-label" id={labelId}>{t("account.nameColor")}</span>
            <span className="menu-label account-color-preview" style={authorStyle({color: chosen, guest: false})}>
                {words.masked(name)}
            </span>
        </div>
        <div className="account-color-swatches" role="group" aria-labelledby={labelId}>
            {NAME_COLORS.map((choice) => swatch(choice.color, choice.hue))}
        </div>
        <p className="account-name-hint">{t("account.nameColorHint")}</p>
        {state.colorFailure && <p className="account-failure" role="alert">{colorMessageOf(state.colorFailure)}</p>}
    </div>
}
