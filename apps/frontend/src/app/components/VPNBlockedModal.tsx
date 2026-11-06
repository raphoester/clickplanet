import Modal from "./Modal.tsx";
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts";
import "./VPNBlockedModal.css"

export default function VPNBlockedModal() {
    const blocked = useRefusedClicks((refused) => refused.vpnBlocked)
    const dismiss = useRefusedClicks((refused) => refused.dismissVPNBlocked)
    if (!blocked) return null

    return <Modal
        title="VPN detected"
        footer={<button type="button" className="button button-action" onClick={dismiss}>Got it</button>}
        onClose={dismiss}>
        <div className="vpn-blocked-text">
            <h3>You can't paint through a VPN 🛡️</h3>
            <p>The server turned down your last click: it arrived from a VPN or proxy address,
                and those don't get to claim tiles.</p>
            <p>Turn yours off and reload to carry on conquering. You can keep watching the
                planet either way — everything else still works.</p>
        </div>
    </Modal>
}
