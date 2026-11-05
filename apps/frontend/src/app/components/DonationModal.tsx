import {useState} from "react";
import Modal from "./Modal.tsx";
import BuyMeACoffee from "./BuyMeACoffee.tsx";
import {COUNTRY_STORAGE_KEY} from "../countryPreference.ts";
import "./DonationModal.css"

const SHOW_PROBABILITY = 0.5

function playedBefore(): boolean {
    try {
        return window.localStorage.getItem(COUNTRY_STORAGE_KEY) !== null
    } catch {
        return false
    }
}

export default function DonationModal() {
    const [isOpen, setIsOpen] = useState(() => playedBefore() && Math.random() > 1 - SHOW_PROBABILITY)
    if (!isOpen) return null

    return <Modal title="Like ClickPlanet?" footer={<BuyMeACoffee/>} onClose={() => setIsOpen(false)}>
        <div className="center-align">
            <img alt="ClickPlanet logo"
                 src="/static/logo.svg"
                 width="64px"
                 height="auto"/>
        </div>
        <div className="donation-text">
            <p>It's free and open source.</p>
            <p>The servers cost money. Each coffee helps pay for them.</p>
        </div>
    </Modal>
}
