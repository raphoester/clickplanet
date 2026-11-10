import {useState} from "react";
import Modal from "./Modal.tsx";
import HoldButton from "./HoldButton.tsx";
import {GAME_RULES} from "./BannedModal.tsx";
import {useWarning} from "../warning/warningStore.ts";
import {WarningAcknowledger} from "../../backends/antibot.ts";
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
        title="One account per player"
        footer={<HoldButton label="Hold to confirm" holdMs={ACKNOWLEDGE_HOLD_MS} disabled={sending} onHeld={acknowledge}/>}>
        <div className="banned-text">
            <p>You played several accounts to get more clicks. Keep to one account, or your accounts will be paused.</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">Read the game rules</a></p>
        </div>
    </Modal>
}
