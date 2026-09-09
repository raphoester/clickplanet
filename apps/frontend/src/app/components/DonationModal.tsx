import {useState} from "react";
import Modal from "./Modal.tsx";
import BuyMeACoffee from "./BuyMeACoffee.tsx";
import "./DonationModal.css"

const SHOW_PROBABILITY = 0.5

export default function DonationModal() {
    const [isOpen, setIsOpen] = useState(() => Math.random() > 1 - SHOW_PROBABILITY)
    if (!isOpen) return null

    return <Modal title="Dear earthlings" footer={<BuyMeACoffee/>} onClose={() => setIsOpen(false)}>
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
    </Modal>
}
