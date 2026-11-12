import {useState} from "react";
import Modal from "./Modal.tsx";
import HoldButton from "./HoldButton.tsx";
import {GAME_RULES} from "./BannedModal.tsx";
import {useWarning} from "../warning/warningStore.ts";
import {WarningAcknowledger} from "../../backends/antibot.ts";
import {t} from "../../i18n/page.ts";
import "./BannedModal.css"

export const ACKNOWLEDGE_HOLD_MS = 2_000

export default function WarningModal({warnings}: {warnings: WarningAcknowledger}) {
    const warned = useWarning((warning) => warning.warned)
    const dismiss = useWarning((warning) => warning.dismiss)
    const [sending, setSending] = useState(false)
    if (!warned) return null

    const acknowledge = () => {
        setSending(true)
        warnings.acknowledgeWarning()
            .then(dismiss, (e: unknown) => console.error("could not acknowledge the warning", e))
            .finally(() => setSending(false))
    }

    return <Modal
        title={t("warning.title")}
        footer={<HoldButton label={t("warning.holdToConfirm")} holdMs={ACKNOWLEDGE_HOLD_MS} disabled={sending} onHeld={acknowledge}/>}>
        <div className="banned-text">
            <p>{t("warning.severalAccounts")}</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">{t("dialog.readRules")}</a></p>
        </div>
    </Modal>
}
