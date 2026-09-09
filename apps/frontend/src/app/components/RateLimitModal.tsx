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
 * Alone among the dialogs, a click on its backdrop does not dismiss it. It
 * appears while the player is mid-burst, so the next click of that burst lands
 * on the backdrop a few milliseconds later and would close the thing unread —
 * the one warning we have, defeated by exactly the behaviour it is about. The ×,
 * the "Got it" button and Escape all still close it: a deliberate act is the
 * point, not a hostage dialog.
 *
 * It says nothing about the exact allowance. The numbers live in the backend's
 * config and are meant to be tunable there; copy quoting them would go stale
 * silently, and the player only needs to know to ease off.
 */
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
