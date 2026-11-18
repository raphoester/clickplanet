import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import type {WorldGrew as Grown} from "../../domain/worldGrew.ts"
import {t} from "../../i18n/page.ts"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import "../titles/TitleUnlocked.css"
import "./WorldGrew.css"

export type WorldGrewProps = {
    grew: Grown
    play: PlaySound
    onClose: () => void
}

// The world's switch of board, once per browser: the title reveal's stage, the globe in place of a title.
export default function WorldGrew({grew, play, onClose}: WorldGrewProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const timesId = useId()
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
             className="title-reveal world-grew"
             role="dialog"
             aria-modal="true"
             aria-labelledby={`${headId} ${timesId}`}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head world-grew-head">{t("world.grew.head")}</h2>
                <div className="world-grew-globe">
                    <div className="title-reveal-rays world-grew-rays" aria-hidden="true"/>
                    <span className="world-grew-art" aria-hidden="true">🌍</span>
                </div>
                <p id={timesId} className="world-grew-times">{t("world.grew.times", {times: grew.times})}</p>
                <p className="title-reveal-rank">{t("world.grew.kept")}</p>
                <div className="title-reveal-actions world-grew-actions">
                    {ready && <button type="button" className="button button-gold" onClick={onClose}>{t("common.close")}</button>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
