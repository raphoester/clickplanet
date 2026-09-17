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

/** Says exactly what goes, because nothing brings it back. */
export default function DeleteAccountModal(props: DeleteAccountModalProps) {
    return <Modal title="Delete your account?"
                  stayOnBackdropClick={props.busy}
                  footer={<div className="account-confirm">
                      <button type="button"
                              className="button button-ghost"
                              disabled={props.busy}
                              onClick={props.onClose}>
                          Cancel
                      </button>
                      <button type="button"
                              className="button account-delete"
                              disabled={props.busy}
                              onClick={props.onConfirm}>
                          Delete
                      </button>
                  </div>}
                  onClose={props.onClose}>
        <div className="account-confirm-text">
            <p>This deletes:</p>
            <ul>
                <li>your account,</li>
                {props.linked.length > 0 && <li>its link to {providerList(props.linked)},</li>}
                <li>your sign-in on all your devices.</li>
            </ul>
            <p>You cannot undo this. You can continue to play as a guest.</p>
            <p>Chat messages and tiles are not linked to your account. They go away on the schedule in
                the <a href="/privacy" target="_blank" rel="noopener noreferrer">privacy policy</a>.</p>
        </div>
    </Modal>
}
