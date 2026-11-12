import {CSSProperties, useId, useState} from "react"
import {TitleTrack} from "../../backends/progression.ts"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {filledOf, leftLabel, stepLabel} from "../titles/titleArt.ts"
import {t} from "../../i18n/page.ts"

const RANK_MEDAL = 48

export default function ProfileTracks({tracks}: {tracks: TitleTrack[]}) {
    const [picked, setPicked] = useState<string>()
    const tabsId = useId()
    const shown = tracks.find((track) => track.id === picked) ?? tracks[0]
    const panelId = `${tabsId}-ranks`

    return <div className="profile-tracks">
        <div className="profile-track-tabs" role="tablist" aria-label={t("profile.titleTracks")} aria-orientation="vertical">
            {tracks.map((track) => <TrackTab key={track.id}
                                             track={track}
                                             id={`${tabsId}-${track.id}`}
                                             panelId={panelId}
                                             selected={track.id === shown.id}
                                             onSelect={() => setPicked(track.id)}/>)}
        </div>
        <TrackRanks track={shown} id={panelId} tabId={`${tabsId}-${shown.id}`}/>
    </div>
}

type TrackTabProps = {
    track: TitleTrack
    id: string
    panelId: string
    selected: boolean
    onSelect: () => void
}

function TrackTab({track, id, panelId, selected, onSelect}: TrackTabProps) {
    const earned = track.steps.filter((step) => step.earned)
    const current = earned.at(-1)
    const next = track.steps.find((step) => !step.earned)
    const from = current?.threshold ?? 0
    const filled = next ? Math.min(1, Math.max(0, (track.progress - from) / (next.threshold - from))) : 1

    return <button type="button"
                   role="tab"
                   id={id}
                   aria-selected={selected}
                   aria-controls={panelId}
                   className="panel-box profile-track-tab"
                   onClick={onSelect}>
        <TitleEmblem title={(current ?? track.steps[0]).title} size={44} locked={!current} ribbon={!!current}/>
        <span className="profile-track-what">
            <span className="menu-label">{t("profile.trackRank", {track: track.name, rank: earned.length, ranks: track.steps.length})}</span>
            <span className="profile-track-rank">{current?.title.name ?? t("profile.noRank")}</span>
            <span className="profile-track-bar" aria-hidden="true">
                <span className="profile-track-bar-filled" style={{width: `${filled * 100}%`}}/>
            </span>
        </span>
    </button>
}

function TrackRanks({track, id, tabId}: {track: TitleTrack, id: string, tabId: string}) {
    const next = track.steps.findIndex((step) => !step.earned)
    const nextStep = next === -1 ? undefined : track.steps[next]

    return <div className="panel-box profile-ranks" role="tabpanel" id={id} aria-labelledby={tabId}>
        <div className="profile-section-head">
            <h3 className="profile-front-title">{track.name}</h3>
            <span className="profile-track-left">
                {nextStep
                    ? leftLabel(track.id, Math.max(0, nextStep.threshold - track.progress), nextStep.title.name, track.progress)
                    : t("profile.everyRank")}
            </span>
        </div>
        <div className="profile-ranks-path" style={{"--rank-medal": `${RANK_MEDAL}px`} as CSSProperties}>
            <span className="profile-ranks-bar" aria-hidden="true">
                <span className="profile-ranks-bar-filled" style={{height: `${filledOf(track) * 100}%`}}/>
            </span>
            <ol className="profile-ranks-list">
                {track.steps.map((step, i) =>
                    <li key={step.title.id}
                        className={`profile-rank${step.earned ? " profile-rank--earned" : ""}${i === next ? " profile-rank--next" : ""}`}
                        aria-current={i === next ? "step" : undefined}>
                        <span className={i === next ? "profile-rank-medal title-pulse" : "profile-rank-medal"}>
                            <TitleEmblem title={step.title} size={RANK_MEDAL} locked={!step.earned} ribbon/>
                        </span>
                        <span className="profile-rank-name">{step.title.name}</span>
                        <span className="profile-rank-threshold">{stepLabel(track.id, step.threshold)}</span>
                    </li>)}
            </ol>
        </div>
    </div>
}
