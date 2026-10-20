import Modal from "./Modal.tsx";
import {describeEnd} from "../../domain/endTime.ts";
import "./BannedModal.css"

export const GAME_RULES = "/terms#play-fair"

export type BannedModalProps = {
    until: number
    onClose: () => void
}

export default function BannedModal(props: BannedModalProps) {
    return <Modal
        title="Clicks paused"
        footer={<button type="button" className="button button-action" onClick={props.onClose}>Got it</button>}
        onClose={props.onClose}>
        <div className="banned-text">
            <p>Clicks from your account or network are paused until {describeEnd(props.until, Date.now())}.</p>
            <p><a href={GAME_RULES} target="_blank" rel="noopener noreferrer">Read the game rules</a></p>
        </div>
    </Modal>
}
