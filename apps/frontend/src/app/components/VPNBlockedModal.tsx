import Modal from "./Modal.tsx";
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts";
import {t} from "../../i18n/page.ts";
import "./VPNBlockedModal.css"

export default function VPNBlockedModal() {
    const blocked = useRefusedClicks((refused) => refused.vpnBlocked)
    const dismiss = useRefusedClicks((refused) => refused.dismissVPNBlocked)
    if (!blocked) return null

    return <Modal
        title={t("dialog.vpnTitle")}
        footer={<button type="button" className="button button-action" onClick={dismiss}>{t("dialog.gotIt")}</button>}
        onClose={dismiss}>
        <div className="vpn-blocked-text">
            <h3>{t("dialog.vpnHead")}</h3>
            <p>{t("dialog.vpnWhy")}</p>
            <p>{t("dialog.vpnFix")}</p>
        </div>
    </Modal>
}
