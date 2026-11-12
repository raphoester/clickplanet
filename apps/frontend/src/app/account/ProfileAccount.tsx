import {useId, useState} from "react"
import {Provider} from "../../backends/account.ts"
import {ClipsNameBackend} from "../../backends/clips.ts"
import ClipsSwitch from "../clips/ClipsSwitch.tsx"
import {CheckIcon} from "../components/icons.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import {messageOf, providerName} from "./authMessages.ts"
import {ColorPicker, UsernameForm} from "./AccountPanel.tsx"
import EmailSignIn from "./EmailSignIn.tsx"
import {t} from "../../i18n/page.ts"

type Ready = Extract<AccountState, {kind: "ready"}>

export type ProfileAccountProps = {
    state: Ready
    store: AccountStore
    clips: ClipsNameBackend
    onDelete: () => void
}

// "email" reads inside a sentence ("Signed in with Google and email"); as a chip of its own it is a name.
const chipName = (provider: Provider) => provider === "email" ? t("account.emailChip") : providerName(provider)

// The account beside a signed-in player's own card: the same settings as the phone's Account tab, made short.
// The providers are chips, a link is a dashed chip, the email form opens on its chip, and leaving is a row of
// small buttons at the foot.
export default function ProfileAccount({state, store, clips, onDelete}: ProfileAccountProps) {
    const [linkingEmail, setLinkingEmail] = useState(false)
    const titleId = useId()
    const providersId = useId()
    const busy = state.busy !== undefined
    const linked = state.me.linked
    const toLink = state.offered.filter((p) => !linked.includes(p))
    // CrazyGames signs the player in and out itself.
    const leavable = !linked.includes("crazygames")

    const link = (provider: Provider) => {
        if (provider === "email") {
            setLinkingEmail((was) => !was)
            return
        }
        void (provider === "crazygames" ? store.signIn(provider) : store.link(provider))
    }

    return <section className="panel-box profile-account" aria-labelledby={titleId} aria-busy={busy}>
        <h3 className="profile-section-title" id={titleId}>{t("account.title")}</h3>

        <UsernameForm key={state.username ?? ""} state={state} store={store}/>
        {state.username !== undefined && <ColorPicker name={state.username} state={state} store={store}/>}

        <div className="profile-providers">
            <span className="menu-label" id={providersId}>{t("account.signedInWithLabel")}</span>
            <ul className="profile-provider-chips" aria-labelledby={providersId}>
                {linked.map((provider) => <li key={provider} className="profile-provider-chip">
                    <CheckIcon size={14}/>{chipName(provider)}
                </li>)}
                {toLink.map((provider) => <li key={provider}>
                    <button type="button"
                            className="profile-provider-chip profile-provider-chip--link"
                            aria-label={t("account.link", {provider: providerName(provider)})}
                            aria-expanded={provider === "email" ? linkingEmail : undefined}
                            disabled={busy}
                            onClick={() => link(provider)}>
                        + {chipName(provider)}
                    </button>
                </li>)}
            </ul>
            {linkingEmail && toLink.includes("email") && <EmailSignIn state={state} store={store} intent="link"/>}
        </div>

        {state.username !== undefined && <ClipsSwitch key={state.accountId} clips={clips}/>}

        {state.failure && <p className="account-failure" role="alert">{messageOf(state.failure)}</p>}

        <div className="profile-account-foot">
            {leavable && <>
                <button type="button" className="account-text-button" disabled={busy} onClick={() => void store.signOut()}>
                    {t("account.signOut")}
                </button>
                <button type="button" className="account-text-button" disabled={busy} onClick={() => void store.signOutEverywhere()}>
                    {t("account.signOutEverywhere")}
                </button>
            </>}
            <button type="button" className="account-text-button account-delete" disabled={busy} onClick={onDelete}>
                {t("account.delete")}
            </button>
            <a className="profile-account-legal" href="/privacy" target="_blank" rel="noopener noreferrer">{t("account.privacyPolicy")}</a>
        </div>
    </section>
}
