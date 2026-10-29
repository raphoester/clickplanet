import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {FlagMission} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import {partsOf} from "./orderParts.tsx"
import "../titles/TitleUnlocked.css"
import "./FlagMissionDone.css"

export type FlagMissionDoneProps = {
    mission: FlagMission
    country: string
    play: PlaySound
    onClose: () => void
}

export default function FlagMissionDone({mission, country, play, onClose}: FlagMissionDoneProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const nameId = useId()
    const [ready, setReady] = useState(false)

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
             className="title-reveal flag-mission-done"
             role="dialog"
             aria-modal="true"
             aria-labelledby={`${headId} ${nameId}`}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head">Bravo!</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <div className="title-reveal-wave" aria-hidden="true"/>
                    <span className="title-reveal-float">
                        <span className="flag-mission-done-disc"><CountryFlag code={country}/></span>
                    </span>
                </div>
                <p id={nameId} className="title-reveal-name">{Countries.get(country)?.name ?? country}</p>
                <p className="title-reveal-rank">Flag mission done: {partsOf(mission.order).line}</p>
                <p className="title-reveal-rank flag-mission-done-points">
                    +{mission.points} season {mission.points === 1 ? "point" : "points"}
                </p>
                <div className="title-reveal-actions">
                    {ready && <button type="button" className="button" onClick={onClose}>Close</button>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
