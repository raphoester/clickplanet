import {useState} from "react"
import {PlayerTitle} from "../../backends/player.ts"
import Modal from "../components/Modal.tsx"
import TitleEmblem from "./TitleEmblem.tsx"
import {rankLine} from "./titleArt.ts"
import "./Titles.css"

export type TitleUnlockedProps = {
    title: PlayerTitle
    onWear?: (id: string) => Promise<unknown>
    onClose: () => void
}

export default function TitleUnlocked({title, onWear, onClose}: TitleUnlockedProps) {
    const [wearing, setWearing] = useState(false)
    const line = rankLine(title)

    const wear = onWear && (() => {
        setWearing(true)
        onWear(title.id).then(onClose, (e) => {
            console.error("Could not wear the title", e)
            setWearing(false)
        })
    })

    return <Modal title="New title unlocked"
                  className="title-unlocked"
                  onClose={onClose}
                  footer={<div className="title-unlocked-actions">
                      <button type="button" className="button" onClick={onClose}>Close</button>
                      {wear && <button type="button" className="button button-gold title-wear-button" disabled={wearing} onClick={wear}>
                          Wear it
                      </button>}
                  </div>}>
        <div className="title-unlocked-stage">
            <div className="title-rays" aria-hidden="true"/>
            <span className="title-unlocked-medal"><TitleEmblem title={title} size={160} ribbon/></span>
        </div>
        <p className="title-unlocked-name">{title.name}</p>
        {line && <p className="title-unlocked-rank">{line}</p>}
    </Modal>
}
