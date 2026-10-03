import {useEffect, useRef} from "react"
import {TitleTrack} from "../../backends/player.ts"
import TitleEmblem from "./TitleEmblem.tsx"
import {filledOf, leftLabel, stepLabel} from "./titleArt.ts"
import "./Titles.css"

const NODE = 104
const GAP = 16
const PAD = 24

export default function TrackPath({track}: {track: TitleTrack}) {
    const scroller = useRef<HTMLDivElement>(null)
    const next = track.steps.findIndex((step) => !step.earned)
    const focus = next === -1 ? track.steps.length - 1 : next
    const span = (track.steps.length - 1) * (NODE + GAP)

    useEffect(() => {
        const element = scroller.current
        if (!element) return
        element.scrollLeft = PAD + focus * (NODE + GAP) + NODE / 2 - element.clientWidth / 2
    }, [focus])

    const nextStep = next === -1 ? undefined : track.steps[next]
    return <section className="panel-box track-path" aria-label={track.name}>
        <div className="track-path-head">
            <h3 className="track-path-name">{track.name}</h3>
            {nextStep && <span className="track-path-left">
                {leftLabel(track.id, Math.max(0, nextStep.threshold - track.progress), nextStep.title.name, track.progress)}
            </span>}
        </div>
        <div className="track-path-scroll" ref={scroller}>
            <div className="track-path-steps" style={{width: track.steps.length * NODE + (track.steps.length - 1) * GAP}}>
                <div className="track-path-bar" style={{left: PAD + NODE / 2, width: span}} aria-hidden="true">
                    <div className="track-path-bar-filled" style={{width: `${filledOf(track) * 100}%`}}/>
                </div>
                <ol className="track-path-list">
                    {track.steps.map((step, i) =>
                        <li key={step.title.id}
                            className={`track-path-step${step.earned ? " track-path-step-earned" : ""}${i === next ? " track-path-step-next" : ""}`}
                            aria-current={i === next ? "step" : undefined}>
                            <span className={i === next ? "track-path-medal title-pulse" : "track-path-medal"}>
                                <TitleEmblem title={step.title} size={56} locked={!step.earned} ribbon/>
                            </span>
                            <span className="track-path-step-name">{step.title.name}</span>
                            <span className="track-path-step-threshold">{stepLabel(track.id, step.threshold)}</span>
                        </li>)}
                </ol>
            </div>
        </div>
    </section>
}
