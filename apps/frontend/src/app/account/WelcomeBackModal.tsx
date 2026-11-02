import Modal from "../components/Modal.tsx"
import "./Account.css"

export type WelcomeBackModalProps = {
    onClose: () => void
}

// A sign-in brought back an account its player had asked to delete: it is no longer on its way out.
export default function WelcomeBackModal(props: WelcomeBackModalProps) {
    return <Modal
        title="Welcome back!"
        footer={<button type="button" className="button button-action" onClick={props.onClose}>Play</button>}
        onClose={props.onClose}>
        <div className="account-confirm-text">
            <p>Your account is no longer going to be deleted. It is back as you left it, and the other players see it
                again.</p>
        </div>
    </Modal>
}
