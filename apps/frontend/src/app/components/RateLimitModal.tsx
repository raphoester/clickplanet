import Modal from "./Modal.tsx";
import "./RateLimitModal.css"

export type RateLimitModalProps = {
    onClose: () => void
}

/**
 * Shown when the server refuses a click as too fast. A modal rather than a
 * toast on purpose: its backdrop covers the globe, so the clicking stops while
 * it is up, which is the whole point of telling the player at all.
 *
 * It says nothing about the exact allowance. The numbers live in the backend's
 * config and are meant to be tunable there; copy quoting them would go stale
 * silently, and the player only needs to know to ease off.
 */
export default function RateLimitModal(props: RateLimitModalProps) {
    return <Modal
        title="Slow down!"
        footer={<button type="button" className="button" onClick={props.onClose}>Got it</button>}
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
