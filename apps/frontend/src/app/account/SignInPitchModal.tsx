import {PROVIDER_NAMES} from "../../backends/account.ts"
import {GUEST_PREFIX} from "../../backends/chat.ts"
import {factor} from "../../domain/clickPrice.ts"
import {MIN_STREAK_SHOWN} from "../../domain/streak.ts"
import Modal from "../components/Modal.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import {messageOf} from "./authMessages.ts"
import ProviderButton from "./ProviderButton.tsx"
import EmailSignIn from "./EmailSignIn.tsx"
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
    const times = `${factor(multiplier)}×`

    return <Modal title="Sign in and stand out" onClose={onClose}>
        <div className="account-panel sign-in-pitch" aria-busy={busy}>
            <ul className="sign-in-pitch-perks">
                <li>
                    <strong>Click {times} faster.</strong> Your clicks refill {times} as fast, in a bank of
                    your own. Guests on one network share one bank.
                </li>
                <li>
                    <strong>Your name.</strong> Pick a username. The chat and the player list show it in
                    place of {GUEST_PREFIX}….
                </li>
                <li>
                    <strong>Your color.</strong> Choose the color of your name. Everyone sees it. Guests
                    are grey.
                </li>
                <li>
                    <strong>Your place on the board.</strong> Players with a name are listed first among
                    the players online.
                </li>
                <li>
                    <strong>Your streak flame.</strong> Play {MIN_STREAK_SHOWN} days in a row and a flame
                    shows beside your name. Your stats follow you on every device.
                </li>
            </ul>
            <p className="account-text sign-in-pitch-small">It is free. You do not need an account to play.</p>

            {state.offered.filter((p) => p !== "email").map((provider) => <ProviderButton key={provider}
                                                                                       provider={provider}
                                                                                       label={`Sign in with ${PROVIDER_NAMES[provider]}`}
                                                                                       disabled={busy}
                                                                                       onClick={() => void store.signIn(provider)}/>)}

            {state.offered.includes("email") && <EmailSignIn state={state} store={store} intent="signIn"/>}

            {state.failure && <p className="account-failure" role="alert">{messageOf(state.failure)}</p>}

            <p className="account-legal">
                <a href="/privacy" target="_blank" rel="noopener noreferrer">Privacy policy</a>
            </p>
        </div>
    </Modal>
}
