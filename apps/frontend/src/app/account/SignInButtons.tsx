import {AccountState, AccountStore} from "./accountStore.ts"
import {messageOf, providerName} from "./authMessages.ts"
import ProviderButton from "./ProviderButton.tsx"
import EmailSignIn from "./EmailSignIn.tsx"
import {t} from "../../i18n/page.ts"

type Ready = Extract<AccountState, {kind: "ready"}>

export default function SignInButtons({state, store}: {state: Ready, store: AccountStore}) {
    const busy = state.busy !== undefined

    return <>
        {state.offered.filter((p) => p !== "email").map((provider) => <ProviderButton key={provider}
                                                                                   provider={provider}
                                                                                   label={t("account.signInWith", {provider: providerName(provider)})}
                                                                                   disabled={busy}
                                                                                   onClick={() => void store.signIn(provider)}/>)}

        {state.offered.includes("email") && <EmailSignIn state={state} store={store} intent="signIn"/>}

        {state.failure && <p className="account-failure" role="alert">{messageOf(state.failure)}</p>}

        <p className="account-legal">
            <a href="/privacy" target="_blank" rel="noopener noreferrer">{t("account.privacyPolicy")}</a>
        </p>
    </>
}
