import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {Appearance, Platform} from "../../backends/clips.ts"
import {watchLinksOf} from "../../domain/clipAppearances.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {PlayIcon, TikTokIcon, YouTubeIcon} from "../components/icons.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import "../titles/TitleUnlocked.css"
import "./ClipNotice.css"

export type ClipNoticeProps = {
    appearance: Appearance
    play: PlaySound
    onClose: () => void
}

export default function ClipNotice({appearance, play, onClose}: ClipNoticeProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const titleId = useId()
    const [ready, setReady] = useState(false)
    const links = watchLinksOf(appearance)

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
             className="title-reveal clip-notice"
             role="dialog"
             aria-modal="true"
             aria-labelledby={`${headId} ${titleId}`}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head clip-notice-head">You are in a clip!</h2>
                <div className="title-reveal-medal">
                    <div className="title-reveal-rays" aria-hidden="true"/>
                    <div className="title-reveal-wave" aria-hidden="true"/>
                    <span className="title-reveal-float">
                        <span className="clip-notice-disc" aria-hidden="true"><PlayIcon size={110}/></span>
                    </span>
                </div>
                <p id={titleId} className="title-reveal-name clip-notice-title">{appearance.title}</p>
                <div className="title-reveal-actions clip-notice-actions">
                    {ready && <>
                        {links.map((link, i) => <a key={link.platform}
                                                   className={`button ${i === 0 ? "button-action" : "button-secondary"}`}
                                                   href={link.url}
                                                   target="_blank"
                                                   rel="noopener noreferrer"
                                                   onClick={onClose}>
                            <PlatformIcon platform={link.platform}/>
                            {link.label}
                        </a>)}
                        <button type="button" className="button" onClick={onClose}>Close</button>
                    </>}
                </div>
            </div>
            <div className="title-reveal-flash" aria-hidden="true"/>
        </div>,
        document.body,
    )
}

function PlatformIcon({platform}: {platform: Platform}) {
    switch (platform) {
        case "tiktok":
            return <TikTokIcon size={22}/>
        case "youtube":
            return <YouTubeIcon size={22}/>
    }
}
