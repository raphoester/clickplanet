import {useState} from "react";
import Modal from "./Modal.tsx";

export default function MapUpdateModal() {
    const [isOpen, setIsOpen] = useState(true)
    if (!isOpen) return null

    const close = () => setIsOpen(false)
    return <Modal
        title="Map update in progress"
        footer={<button type="button" className="button button-action" onClick={close}>Got it</button>}
        onClose={close}>
        <p>We are updating the map. The game is down for a few minutes.</p>
        <p>Clicks do not count right now. When the update is done, reload the page.</p>
    </Modal>
}
