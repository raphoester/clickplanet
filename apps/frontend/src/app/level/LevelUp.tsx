import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {PlayerLevel} from "../../backends/progression.ts"
import {nextUnlock, unlockedBy} from "../../domain/level.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import BonusName from "./BonusName.tsx"
import {useLevel} from "./levelStore.ts"
import "../titles/TitleUnlocked.css"
import "./Level.css"

const SPARKS = 36

const COLORS = ["var(--gold)", "var(--gold-light)", "var(--cream)", "var(--gold-shade)"]

const SHAPES = ["title-reveal-spark", "title-reveal-spark title-reveal-spark-long", "title-reveal-spark title-reveal-spark-dot"]

type Spark = {x: number, y: number, spin: number, delay: number, color: string, shape: string}

function sparks(): Spark[] {
    return Array.from({length: SPARKS}, (_, i) => {
        const angle = (i / SPARKS + Math.random() * 0.02) * 2 * Math.PI
        const reach = 0.5 + Math.random() * 0.5
        return {
            x: Math.cos(angle) * reach,
            y: Math.sin(angle) * reach,
            spin: (Math.random() * 2 - 1) * 540,
            delay: Math.random() * 90,
            color: COLORS[i % COLORS.length],
            shape: SHAPES[i % SHAPES.length],
        }
    })
}

export type LevelUpProps = {
    level: PlayerLevel
    // The level shown before: none for the account's first, which names no kind unlocked.
    from?: number
    play: PlaySound
    onClose: () => void
}

// The title unlock's stage, with the level's number where the medal lands, the kinds of bonus the raise unlocked and the
// next one. Nothing closes it before it has landed.
export default function LevelUp({level, from, play, onClose}: LevelUpProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const nameId = useId()
    const [shown] = useState(sparks)
    const [ready, setReady] = useState(false)
    const unlocks = useLevel((store) => store.unlocks)
    const unlocked = from === undefined ? [] : unlockedBy(unlocks, from, level.level)
    const next = nextUnlock(unlocks, level.level)

    useEffect(() => {
        play("title")
        const timer = setTimeout(() => setReady(true), TITLE_REVEAL.ready * 1000)
        return () => clearTimeout(timer)
    }, [play])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    const timing = {"--impact": `${TITLE_REVEAL.impact * 1000}ms`, "--metal": "var(--gold)"} as CSSProperties

    return createPortal(
        <div ref={dialog}
             className="title-reveal level-up"
             role="dialog"
             aria-modal="true"
             aria-labelledby={`${headId} ${nameId}`}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head">{t("level.up")}</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <div className="title-reveal-wave" aria-hidden="true"/>
                    <div className="title-reveal-sparks" aria-hidden="true">
                        {shown.map((spark, i) => <i key={i}
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
                        <span className="title-reveal-spin">
                            <span className="level-up-badge" aria-hidden="true">{translator().number(level.level)}</span>
                        </span>
                    </span>
                </div>
                <p id={nameId} className="title-reveal-name">{t("level.level", {level: level.level})}</p>
                {unlocked.map(({kind}) => <p key={kind} className="title-reveal-rank level-up-unlocked">
                    {rich("level.unlockedNow", {bonus: <BonusName kind={kind}/>})}
                </p>)}
                {next && <p className="title-reveal-rank level-up-next">
                    {rich("level.nextUnlock", {bonus: <BonusName kind={next.kind}/>, level: next.level})}
                </p>}
                <div className="title-reveal-actions">
                    {ready && <button type="button" className="button button-gold" onClick={onClose}>{t("common.ok")}</button>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
