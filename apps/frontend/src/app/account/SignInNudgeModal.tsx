import {factor} from "../../domain/clickPrice.ts"
import Modal from "../components/Modal.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import SignInButtons from "./SignInButtons.tsx"
import "./Account.css"

type Ready = Extract<AccountState, {kind: "ready"}>

export type SignInNudgeModalProps = {
    state: Ready
    store: AccountStore
    multiplier: number
    fullBank: boolean
    onClose: () => void
}

export default function SignInNudgeModal({state, store, multiplier, fullBank, onClose}: SignInNudgeModalProps) {
    return <Modal title="Out of clicks" onClose={onClose}>
        <div className="account-panel sign-in-pitch" aria-busy={state.busy !== undefined}>
            <ul className="sign-in-pitch-perks">
                {fullBank && <li><strong>A full bank now</strong></li>}
                <li><strong>Clicks refill {factor(multiplier)}× faster</strong></li>
                <li><strong>Your land</strong> on every device</li>
            </ul>

            <SignInButtons state={state} store={store}/>
        </div>
    </Modal>
}
