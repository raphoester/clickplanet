import {factor} from "../../domain/clickPrice.ts"
import Modal from "../components/Modal.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import SignInButtons from "./SignInButtons.tsx"
import {Nudge} from "./useSignInNudge.ts"
import {t} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type SignInNudgeModalProps = {
    state: Ready
    store: AccountStore
    nudge: Nudge
    multiplier: number
    fastStart: boolean
    fullBank: boolean
    onClose: () => void
}

export default function SignInNudgeModal({state, store, nudge, multiplier, fastStart, fullBank, onClose}: SignInNudgeModalProps) {
    return <Modal title={t(nudge === "slowed" ? "signIn.keepSpeed" : "signIn.outOfClicks")} onClose={onClose}>
        <div className="account-panel sign-in-pitch" aria-busy={state.busy !== undefined}>
            <ul className="sign-in-pitch-perks">
                {fullBank && <li><strong>{t("signIn.fullBank")}</strong></li>}
                {fastStart
                    ? <li><strong>{t("signIn.keepSpeed")}</strong></li>
                    : <li><strong>{t("signIn.refillFaster", {times: factor(multiplier)})}</strong></li>}
                <li>{rich("signIn.landEverywhere", {strong: (children) => <strong>{children}</strong>})}</li>
            </ul>

            <SignInButtons state={state} store={store}/>
        </div>
    </Modal>
}
