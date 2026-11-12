import {GUEST_PREFIX} from "../../backends/chat.ts"
import {factor} from "../../domain/clickPrice.ts"
import {MIN_STREAK_SHOWN} from "../../domain/streak.ts"
import Modal from "../components/Modal.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import SignInButtons from "./SignInButtons.tsx"
import {t} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {Key, Values} from "../../i18n/translator.ts"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type SignInPitchModalProps = {
    state: Ready
    store: AccountStore
    multiplier: number
    onClose: () => void
}

export default function SignInPitchModal({state, store, multiplier, onClose}: SignInPitchModalProps) {
    const busy = state.busy !== undefined
    const perk = (key: Key, values: Values = {}) =>
        <li>{rich(key, {...values, strong: (children) => <strong>{children}</strong>})}</li>

    return <Modal title={t("signIn.pitchTitle")} onClose={onClose}>
        <div className="account-panel sign-in-pitch" aria-busy={busy}>
            <ul className="sign-in-pitch-perks">
                {perk("signIn.pitchFaster", {times: factor(multiplier)})}
                {perk("signIn.pitchName", {guest: GUEST_PREFIX})}
                {perk("signIn.pitchTeamChat")}
                {perk("signIn.pitchColor")}
                {perk("signIn.pitchBoard")}
                {perk("signIn.pitchStreak", {count: MIN_STREAK_SHOWN})}
            </ul>
            <p className="account-text sign-in-pitch-small">{t("signIn.pitchFree")}</p>

            <SignInButtons state={state} store={store}/>
        </div>
    </Modal>
}
