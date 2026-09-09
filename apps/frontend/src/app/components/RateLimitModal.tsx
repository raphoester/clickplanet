import Modal from "./Modal.tsx";
import "./RateLimitModal.css"

export type RateLimitModalProps = {
    onClose: () => void
}

export default function RateLimitModal(props: RateLimitModalProps) {
    return <Modal
        title="Slow down!"
        footer={<button type="button" className="button" onClick={props.onClose}>Got it</button>}
        stayOnBackdropClick
        onClose={props.onClose}>
        <div className="rate-limit-text">
            <h3>You're clicking too fast 🔥</h3>
            <p>The server turned down your last clicks: too many were arriving from your
                connection at once.</p>
            <p>Wait a couple of seconds and carry on conquering — the tiles the server did
                accept are still yours.</p>
        </div>
    </Modal>
}
