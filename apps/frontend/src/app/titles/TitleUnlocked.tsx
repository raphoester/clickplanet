import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {PlayerTitle} from "../../backends/player.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import TitleEmblem from "./TitleEmblem.tsx"
import {METALS, metalOf, rankLine, ribbonOf} from "./titleArt.ts"
import "./Titles.css"
import "./TitleUnlocked.css"

const SPARKS = 36

type Spark = {x: number, y: number, spin: number, delay: number, color: string, shape: string}

function sparksOf(title: PlayerTitle): Spark[] {
    const colors = ["var(--gold)", "var(--gold-light)", "var(--cream)", METALS[metalOf(title)], ribbonOf(title)]
    const shapes = ["title-reveal-spark", "title-reveal-spark title-reveal-spark-long", "title-reveal-spark title-reveal-spark-dot"]
    return Array.from({length: SPARKS}, (_, i) => {
        const angle = (i / SPARKS + Math.random() * 0.02) * 2 * Math.PI
        const reach = 0.5 + Math.random() * 0.5
        return {
            x: Math.cos(angle) * reach,
            y: Math.sin(angle) * reach,
            spin: (Math.random() * 2 - 1) * 540,
            delay: Math.random() * 90,
            color: colors[i % colors.length],
            shape: shapes[i % shapes.length],
        }
    })
}

export type TitleUnlockedProps = {
    title: PlayerTitle
    play: PlaySound
    onWear?: (id: string) => Promise<unknown>
    onClose: () => void
}

export default function TitleUnlocked({title, play, onWear, onClose}: TitleUnlockedProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const nameId = useId()
    const [sparks] = useState(() => sparksOf(title))
    const [ready, setReady] = useState(false)
    const [wearing, setWearing] = useState(false)
    const line = rankLine(title)

    useEffect(() => {
        play("title")
        const timer = setTimeout(() => setReady(true), TITLE_REVEAL.ready * 1000)
        return () => clearTimeout(timer)
    }, [play])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    const wear = onWear && (() => {
        setWearing(true)
        onWear(title.id).then(onClose, (e) => {
            console.error("Could not wear the title", e)
            setWearing(false)
        })
    })

    const timing = {"--impact": `${TITLE_REVEAL.impact * 1000}ms`, "--metal": METALS[metalOf(title)]} as CSSProperties

    return createPortal(
        <div ref={dialog}
             className="title-reveal"
             role="dialog"
             aria-modal="true"
             aria-labelledby={`${headId} ${nameId}`}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head">New title!</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <div className="title-reveal-wave" aria-hidden="true"/>
                    <div className="title-reveal-sparks" aria-hidden="true">
                        {sparks.map((spark, i) => <i key={i}
                                                     className={spark.shape}
                                                     style={{
                                                         "--x": spark.x,
                                                         "--y": spark.y,
                                                         "--spin": `${spark.spin}deg`,
                                                         "--delay": `${spark.delay}ms`,
                                                         "--spark": spark.color,
                                                     } as CSSProperties}/>)}
                    </div>
                    <span className="title-reveal-float">
                        <span className="title-reveal-spin"><TitleEmblem title={title} size={200} ribbon/></span>
                    </span>
                </div>
                <p id={nameId} className="title-reveal-name">{title.name}</p>
                {line && <p className="title-reveal-rank">{line}</p>}
                <div className="title-reveal-actions">
                    {ready && <>
                        <button type="button" className="button" onClick={onClose}>Close</button>
                        {wear && <button type="button" className="button button-gold title-wear-button" disabled={wearing} onClick={wear}>
                            Wear it
                        </button>}
                    </>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
