import Modal from "../components/Modal.tsx"
import {Provider} from "../../backends/account.ts"
import {providerList} from "./authMessages.ts"
import "./Account.css"

export type DeleteAccountModalProps = {
    linked: Provider[]
    busy: boolean
    onConfirm: () => void
    onClose: () => void
}

export default function DeleteAccountModal(props: DeleteAccountModalProps) {
    return <Modal title="Delete your account?"
                  stayOnBackdropClick={props.busy}
                  footer={<div className="account-confirm">
                      <button type="button"
                              className="button"
                              disabled={props.busy}
                              onClick={props.onClose}>
                          Cancel
                      </button>
                      <button type="button"
                              className="button button-action"
                              disabled={props.busy}
                              onClick={props.onConfirm}>
                          Delete
                      </button>
                  </div>}
                  onClose={props.onClose}>
        <div className="account-confirm-text">
            <p>You are signed out on all your devices, and the other players no longer see your account.</p>
            <p>Changed your mind? Sign in again within 30 days, and your account comes back as you left it.</p>
            <p>After 30 days, your account is deleted for good{props.linked.length > 0 &&
                <>, with its link to {providerList(props.linked)}</>}. You can continue to play as a guest.</p>
            <p>The tiles you took stay, with no account. Chat messages are not linked to your account. They
                go away on the schedule in the <a href="/privacy" target="_blank" rel="noopener noreferrer">privacy
                    policy</a>.</p>
        </div>
    </Modal>
}
