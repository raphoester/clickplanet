import Modal from "./Modal.tsx";
import {describeEnd} from "../../domain/endTime.ts";
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts";
import "./BannedModal.css"

export const GAME_RULES = "/terms#play-fair"

export default function BannedModal() {
    const until = useRefusedClicks((refused) => refused.bannedUntil)
    const dismiss = useRefusedClicks((refused) => refused.dismissBanned)
    if (until === undefined) return null

    return <Modal
        title="Clicks paused"
        footer={<button type="button" className="button button-action" onClick={dismiss}>Got it</button>}
        onClose={dismiss}>
        <div className="banned-text">
            <p>Clicks from your account or network are paused until {describeEnd(until, Date.now())}.</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">Read the game rules</a></p>
        </div>
    </Modal>
}
