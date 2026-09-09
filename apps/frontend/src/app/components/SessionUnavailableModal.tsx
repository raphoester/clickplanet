import Modal from "./Modal.tsx";
import "./SessionUnavailableModal.css"

export type SessionUnavailableModalProps = {
    onClose: () => void
}

export default function SessionUnavailableModal(props: SessionUnavailableModalProps) {
    return <Modal
        title="Couldn't verify your browser"
        footer={<button type="button" className="button" onClick={props.onClose}>Got it</button>}
        onClose={props.onClose}>
        <div className="session-unavailable-text">
            <h3>That click didn't go through 🤖</h3>
            <p>Before you can paint, the game asks your browser to prove it's a browser. That
                check didn't complete, so the server turned the click down.</p>
            <p>Reloading the page usually fixes it. If it keeps happening, an ad blocker or a
                privacy extension is probably blocking <code>challenges.cloudflare.com</code> —
                allow it for this site and try again.</p>
        </div>
    </Modal>
}
