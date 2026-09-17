import {PROVIDER_NAMES} from "../../backends/account.ts"
import {factor} from "../../domain/clickPrice.ts"
import Modal from "../components/Modal.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import {messageOf} from "./authMessages.ts"
import ProviderButton from "./ProviderButton.tsx"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type SignInPitchModalProps = {
    state: Ready
    store: AccountStore
    /** What the server multiplies a signed-in account's allowance by. */
    multiplier: number
    onClose: () => void
}

/**
 * What the meter's offer opens: why to sign in, and the buttons that do it.
 * The buttons are the account panel's, so signing in from here is the same flow.
 */
export default function SignInPitchModal({state, store, multiplier, onClose}: SignInPitchModalProps) {
    const busy = state.busy !== undefined
    const times = `${factor(multiplier)}×`

    return <Modal title={`Click ${times} faster`} onClose={onClose}>
        <div className="account-panel sign-in-pitch" aria-busy={busy}>
            <p className="account-text">
                Sign in and you get {times} the clicks in hand, refilling {times} as fast. It is free, and your
                stats and username follow you on every device.
            </p>
            <p className="account-text sign-in-pitch-small">You do not need an account to play.</p>

            {state.offered.map((provider) => <ProviderButton key={provider}
                                                             provider={provider}
                                                             label={`Sign in with ${PROVIDER_NAMES[provider]}`}
                                                             disabled={busy}
                                                             onClick={() => void store.signIn(provider)}/>)}

            {state.failure && <p className="account-failure" role="alert">{messageOf(state.failure)}</p>}

            <p className="account-legal">
                <a href="/privacy" target="_blank" rel="noopener noreferrer">Privacy policy</a>
            </p>
        </div>
    </Modal>
}
