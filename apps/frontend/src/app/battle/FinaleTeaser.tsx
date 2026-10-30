import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {Finale, finaleWhenLine} from "../../domain/battle.ts"
import {timeLeftTo} from "../../domain/seasonClock.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {SwordsIcon} from "../components/icons.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {FinaleFacts} from "../season/FinaleFacts.tsx"
import {useNow} from "../season/useNow.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import "../season/Season.css"
import "../titles/TitleUnlocked.css"
import "./Battle.css"

export type FinaleTeaserProps = {
    finale: Finale
    play: PlaySound
    onClose: () => void
}

export default function FinaleTeaser({finale, play, onClose}: FinaleTeaserProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const [ready, setReady] = useState(false)
    const now = useNow()

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
             className="title-reveal battle-reveal finale-teaser"
             data-mood="battle"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head">Final Battle</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <div className="title-reveal-wave" aria-hidden="true"/>
                    <span className="title-reveal-float">
                        <span className="battle-reveal-disc" aria-hidden="true"><SwordsIcon size={80}/></span>
                    </span>
                </div>
                <p className="battle-brief-times finale-teaser-left" role="timer">
                    <span>Starts in {timeLeftTo(finale.startsAt, now)}</span>
                </p>
                <p className="title-reveal-rank">{finaleWhenLine(finale)}</p>
                <div className="panel-box finale-teaser-facts">
                    <FinaleFacts finale={finale}/>
                </div>
                <div className="title-reveal-actions">
                    {ready && <button type="button" className="button" onClick={onClose}>To the globe</button>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
