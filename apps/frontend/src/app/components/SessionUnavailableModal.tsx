import Modal from "./Modal.tsx";
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts";
import {t} from "../../i18n/page.ts";
import {rich} from "../../i18n/rich.tsx";
import "./SessionUnavailableModal.css"

export default function SessionUnavailableModal() {
    const unavailable = useRefusedClicks((refused) => refused.sessionUnavailable)
    const dismiss = useRefusedClicks((refused) => refused.dismissSessionUnavailable)
    if (!unavailable) return null

    return <Modal
        title={t("dialog.browserCheckTitle")}
        footer={<button type="button" className="button button-action" onClick={dismiss}>{t("dialog.gotIt")}</button>}
        onClose={dismiss}>
        <div className="session-unavailable-text">
            <h3>{t("dialog.browserCheckHead")}</h3>
            <p>{t("dialog.browserCheckWhy")}</p>
            <p>{rich("dialog.browserCheckFix", {host: <code>challenges.cloudflare.com</code>})}</p>
        </div>
    </Modal>
}
