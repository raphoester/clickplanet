import Modal from "../components/Modal.tsx"
import {VideoIcon} from "../components/icons.tsx"
import {AdRefill} from "./useAdRefill.ts"
import {adRefillLine, WATCH_AD} from "./adRefillText.ts"
import "./AdRefillModal.css"

export type AdRefillModalProps = {
    adRefill: AdRefill
    countryId: string
    onClose: () => void
}

export default function AdRefillModal({adRefill, countryId, onClose}: AdRefillModalProps) {
    const {state} = adRefill
    const playing = state.kind === "playing"
    const watchable = state.kind === "ready" || state.kind === "unavailable" || playing
    const close = () => {
        adRefill.seen()
        onClose()
    }

    return <Modal title={state.kind === "refilled" ? "A full bank" : "Out of clicks"} stayOnBackdropClick={playing} onClose={close}>
        <div className="ad-refill" aria-busy={playing}>
            <p className="ad-refill-text">{adRefillLine(state)}</p>
            <div className="ad-refill-choices">
                {state.kind === "refilled"
                    ? <button type="button" className="button button-action ad-refill-choice" onClick={close}>Play</button>
                    : <>
                        {watchable && <button type="button"
                                              className="button ad-refill-choice"
                                              disabled={playing}
                                              onClick={() => adRefill.watch(countryId)}>
                            <VideoIcon/>
                            <span>{WATCH_AD}</span>
                        </button>}
                        <button type="button" className="button ad-refill-choice" disabled={playing} onClick={close}>No thanks</button>
                    </>}
            </div>
        </div>
    </Modal>
}
