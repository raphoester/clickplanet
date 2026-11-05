import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {Battle, battleEndsLine, battleNameOf} from "../../domain/battle.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import SwordsArt from "../components/SwordsArt.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import "../titles/TitleUnlocked.css"
import "./Battle.css"

export type BattleRevealProps = {
    battle: Battle
    play: PlaySound
    onClose: () => void
}

export default function BattleReveal({battle, play, onClose}: BattleRevealProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const [ready, setReady] = useState(false)
    const [openedAt] = useState(Date.now)

    useEffect(() => {
        play("title")
        const timer = setTimeout(() => setReady(true), TITLE_REVEAL.ready * 1000)
        return () => clearTimeout(timer)
    }, [play])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    const timing = {"--impact": `${TITLE_REVEAL.impact * 1000}ms`} as CSSProperties

    return createPortal(
        <div ref={dialog}
             className="title-reveal battle-reveal"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head">{battleNameOf(battle)}!</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <div className="title-reveal-wave" aria-hidden="true"/>
                    <span className="title-reveal-float">
                        <span className="battle-reveal-disc" aria-hidden="true"><SwordsArt/></span>
                    </span>
                </div>
                {battle.description && <p className="title-reveal-rank battle-reveal-text">{battle.description}</p>}
                <p className="title-reveal-rank">{battleEndsLine(battle, openedAt)}</p>
                <div className="title-reveal-actions">
                    {ready && <button type="button" className="button" onClick={onClose}>To the globe</button>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
