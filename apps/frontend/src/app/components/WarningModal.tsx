import Modal from "./Modal.tsx";
import {GAME_RULES} from "./BannedModal.tsx";
import {useWarning} from "../warning/warningStore.ts";
import {WarningAcknowledger} from "../../backends/antibot.ts";
import "./BannedModal.css"

export default function WarningModal({warnings}: {warnings: WarningAcknowledger}) {
    const warned = useWarning((warning) => warning.warned)
    const dismiss = useWarning((warning) => warning.dismiss)
    if (!warned) return null

    const acknowledge = () => {
        warnings.acknowledgeWarning().then(dismiss, (e: unknown) => console.error("could not acknowledge the warning", e))
    }

    return <Modal
        title="One account per player"
        footer={<button type="button" className="button button-action" onClick={acknowledge}>Got it</button>}
        onClose={dismiss}>
        <div className="banned-text">
            <p>You played several accounts to get more clicks. Keep to one account, or your accounts will be paused.</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">Read the game rules</a></p>
        </div>
    </Modal>
}
