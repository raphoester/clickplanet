import Modal from "./Modal.tsx";
import "./VPNBlockedModal.css"

export type VPNBlockedModalProps = {
    onClose: () => void
}

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
