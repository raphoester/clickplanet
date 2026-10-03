import {useId, useState} from "react"
import {PlayerTitle} from "../../backends/player.ts"
import Modal from "../components/Modal.tsx"
import TitleEmblem from "./TitleEmblem.tsx"
import {METALS, metalOf, rankLine} from "./titleArt.ts"
import "./Titles.css"

const RAYS = Array.from({length: 15}, (_, i) => {
    const angle = i * 24 * Math.PI / 180
    const spread = 6 * Math.PI / 180
    const point = (a: number) => `${160 + 158 * Math.cos(a)} ${160 + 158 * Math.sin(a)}`
    return `M160 160 L${point(angle - spread)} L${point(angle + spread)} Z`
})

export type TitleUnlockedProps = {
    title: PlayerTitle
    onWear?: (id: string) => Promise<unknown>
    onClose: () => void
}

export default function TitleUnlocked({title, onWear, onClose}: TitleUnlockedProps) {
    const fade = `rays-${useId().replace(/:/g, "")}`
    const [wearing, setWearing] = useState(false)
    const [light, mid, dark] = METALS[metalOf(title)]
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
                      <button type="button" className="button button-ghost" onClick={onClose}>Close</button>
                      {wear && <button type="button" className="button title-wear-button" disabled={wearing} onClick={wear}>
                          Wear it
                      </button>}
                  </div>}>
        <div className="title-unlocked-stage">
            <svg className="title-rays" width="320" height="320" viewBox="0 0 320 320" aria-hidden="true">
                <defs>
                    <radialGradient id={fade} cx="160" cy="160" r="158" gradientUnits="userSpaceOnUse">
                        <stop offset="0%" stopColor={mid} stopOpacity="0.45"/>
                        <stop offset="100%" stopColor={mid} stopOpacity="0"/>
                    </radialGradient>
                </defs>
                {RAYS.map((ray) => <path key={ray} d={ray} fill={`url(#${fade})`}/>)}
            </svg>
            <span className="title-unlocked-medal"><TitleEmblem title={title} size={170} ribbon/></span>
        </div>
        <p className="title-unlocked-name"
           style={{color: light, textShadow: `0 3px 0 ${dark}, 0 6px 0 #000000, 0 0 30px ${mid}88`}}>
            {title.name}
        </p>
        {line && <p className="title-unlocked-rank">{line}</p>}
    </Modal>
}
