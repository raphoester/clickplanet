import Modal from "./Modal.tsx";
import "./VPNBlockedModal.css"

export type VPNBlockedModalProps = {
    onClose: () => void
}

/**
 * Shown when the server refuses a click because it came from a VPN or proxy.
 * A modal for the same reason as RateLimitModal: its backdrop covers the globe,
 * so the clicking stops while it is up.
 *
 * The copy is an instruction rather than "wait a moment", which is the one real
 * difference from the throttle: a spent bucket refills in a second, but this
 * refusal stands until the player changes network. Dismissing it therefore only
 * closes the dialog — the next click raises it again, which is correct.
 *
 * It does not say which list matched or how the address was judged. That is the
 * backend's business, it is tunable there, and the player can act on exactly one
 * thing.
 */
export default function VPNBlockedModal(props: VPNBlockedModalProps) {
    return <Modal
        title="VPN detected"
        footer={<button type="button" className="button" onClick={props.onClose}>Got it</button>}
        onClose={props.onClose}>
        <div className="vpn-blocked-text">
            <h3>You can't paint through a VPN 🛡️</h3>
            <p>The server turned down your last click: it arrived from a VPN or proxy address,
                and those don't get to claim tiles.</p>
            <p>Turn yours off and reload to carry on conquering. You can keep watching the
                planet either way — everything else still works.</p>
        </div>
    </Modal>
}
