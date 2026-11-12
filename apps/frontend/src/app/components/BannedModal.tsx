import Modal from "./Modal.tsx";
import {describeEnd} from "../endTime.ts";
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts";
import {t} from "../../i18n/page.ts";
import "./BannedModal.css"

export const GAME_RULES = "/terms#play-fair"

export default function BannedModal() {
    const until = useRefusedClicks((refused) => refused.bannedUntil)
    const dismiss = useRefusedClicks((refused) => refused.dismissBanned)
    if (until === undefined) return null

    return <Modal
        title={t("ban.title")}
        footer={<button type="button" className="button button-action" onClick={dismiss}>{t("dialog.gotIt")}</button>}
        onClose={dismiss}>
        <div className="banned-text">
            <p>{t("ban.pausedUntil", {time: describeEnd(until, Date.now())})}</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">{t("dialog.readRules")}</a></p>
        </div>
    </Modal>
}
