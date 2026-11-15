import {useState} from "react";
import Modal from "./Modal.tsx";
import {GAME_RULES} from "./BannedModal.tsx";
import {useWarning} from "../warning/warningStore.ts";
import {NoticeAcknowledger} from "../../backends/antibot.ts";
import {t} from "../../i18n/page.ts";
import "./BannedModal.css"

export default function NoticeModal({notices}: {notices: NoticeAcknowledger}) {
    const noticed = useWarning((warning) => warning.noticed)
    const warned = useWarning((warning) => warning.warned)
    const dismiss = useWarning((warning) => warning.dismissNotice)
    const [sending, setSending] = useState(false)
    if (!noticed || warned) return null

    const acknowledge = () => {
        setSending(true)
        notices.acknowledgeNotice()
            .then(dismiss, (e: unknown) => console.error("could not acknowledge the notice", e))
            .finally(() => setSending(false))
    }

    return <Modal
        title={t("warning.title")}
        footer={<button type="button" className="button button-action" disabled={sending} onClick={acknowledge}>{t("notice.understand")}</button>}>
        <div className="banned-text">
            <p>{t("notice.sharedConnection")}</p>
            <p><strong>{t("notice.oneAccount")}</strong></p>
            <p>{t("notice.sharing")}</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">{t("dialog.readRules")}</a></p>
        </div>
    </Modal>
}
