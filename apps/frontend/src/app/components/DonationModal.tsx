import {useState} from "react";
import Modal from "./Modal.tsx";
import BuyMeACoffee from "./BuyMeACoffee.tsx";
import "./DonationModal.css"

/** The share of page loads that get the ask. */
const SHOW_PROBABILITY = 0.5

/**
 * The genuine modal in the app: an unprompted interruption, so it takes the
 * whole screen and has to be dismissed.
 *
 * The roll is decided once, when the component mounts. It used to be a bare
 * `Math.random()` in App's returned JSX, which is a side effect in render:
 * React is free to render a component more than once for a single commit, and
 * the modal would appear or vanish on any re-render.
 */
export default function DonationModal() {
    const [isOpen, setIsOpen] = useState(() => Math.random() > 1 - SHOW_PROBABILITY)
    if (!isOpen) return null

    return <Modal title="Dear earthlings" onClose={() => setIsOpen(false)}>
        <div className="center-align">
            <img alt="ClickPlanet logo"
                 src="/static/logo.svg"
                 width="64px"
                 height="auto"/>
        </div>
        <div className="donation-text">
            <h3>Do you like ClickPlanet ?</h3>
            <p>It's free and open-source 🤗</p>
            <p>Sadly, the servers are expensive to run 😭</p>
            <p>Every contribution helps us keep this awesome platform running!</p>
        </div>
        <BuyMeACoffee/>
    </Modal>
}
