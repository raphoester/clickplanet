import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {headlineOf, VideoStory} from "../../domain/video/story.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {nameOf, said} from "../../clip/names.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import "../titles/TitleUnlocked.css"
import "../clips/ClipNotice.css"
import "./VideoMoment.css"

export type VideoMomentProps = {
    story: VideoStory
    play: PlaySound
    onMake: () => void
    onClose: () => void
}

// The flag is losing a country: the moment offers the video that calls for help.
export default function VideoMoment({story, play, onMake, onClose}: VideoMomentProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
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
             className="title-reveal clip-notice video-moment"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head clip-notice-head">{headlineOf(story, {nameOf, said})}</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <span className="title-reveal-float video-moment-flag"><CountryFlag code={story.country}/></span>
                </div>
                <div className="title-reveal-actions clip-notice-actions">
                    {ready && <>
                        <button type="button" className="button button-action" onClick={() => {
                            onClose()
                            onMake()
                        }}>Make my video</button>
                        <button type="button" className="button" onClick={onClose}>Close</button>
                    </>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}
