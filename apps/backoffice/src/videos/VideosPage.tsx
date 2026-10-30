import {Code, ConnectError} from "@connectrpc/connect"
import {useCallback, useEffect, useRef, useState} from "react"
import {putFile} from "../api.ts"
import {ClipsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {Video} from "../gen/grpc/clips/v1/admin_pb.ts"
import {countOf, sizeLabel} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {copyText, Put, removeRefusalOf, VIDEOS_OFF} from "./actions.ts"
import {UploadPanel} from "./UploadPanel.tsx"

const COPIED_MS = 1500

export function VideosPage({clips, put = putFile}: {clips: ClipsAdmin, put?: Put}) {
    // Undefined when videos are off: failureOf would read failed_precondition as a sign-in to start again.
    const list = useCallback(async (signal: AbortSignal) => {
        try {
            return (await clips.listVideos({}, {signal})).videos
        } catch (err) {
            if (ConnectError.from(err).code === Code.FailedPrecondition) return undefined
            throw err
        }
    }, [clips])
    const [videos, again] = useCall(list)
    const [removed, setRemoved] = useState<Video[]>([])

    if (videos.state === "done" && videos.value === undefined) return <p className="muted">{VIDEOS_OFF}</p>

    const shown = videos.state === "done"
        ? (videos.value ?? []).filter(video => !removed.some(gone => gone.videoId === video.videoId))
        : []
    const last = removed[removed.length - 1]

    return (
        <div className="videos">
            <UploadPanel clips={clips} put={put} onKept={again}/>
            <section className="videos-list" aria-label="Videos">
                <div className="toolbar">
                    <span className="muted">{videos.state === "done" ? countOf(shown.length, "video") : ""}</span>
                    <button type="button" className="button" onClick={again}>Reload</button>
                </div>
                {last !== undefined && <p className="action-said" role="status">Removed {last.name}.</p>}
                {videos.state === "loading" && <p className="muted">Loading…</p>}
                {videos.state === "failed" && <FailureNotice failure={videos} onRetry={again}/>}
                {videos.state === "done" && (
                    <ol className="cards video-grid">
                        {shown.map(video => (
                            <VideoCard
                                key={video.videoId}
                                clips={clips}
                                video={video}
                                onRemoved={() => setRemoved(was => [...was, video])}
                            />
                        ))}
                    </ol>
                )}
            </section>
        </div>
    )
}

function VideoCard({clips, video, onRemoved}: {clips: ClipsAdmin, video: Video, onRemoved: () => void}) {
    const [unplayable, setUnplayable] = useState(false)

    return (
        <li className="card video">
            <h3 className="video-name">{video.name}</h3>
            <p className="muted video-facts">{sizeLabel(video.sizeBytes)} · <When at={video.keptAt}/></p>
            {unplayable
                ? <p className="attachment-unshown muted">This browser cannot play it, or its link expired: Reload.</p>
                : (
                    // A season-end video has no captions to give it.
                    // eslint-disable-next-line jsx-a11y/media-has-caption
                    <video
                        className="video-player"
                        src={video.viewUrl}
                        aria-label={video.name}
                        controls
                        playsInline
                        preload="metadata"
                        onError={() => setUnplayable(true)}
                    />
                )}
            {video.caption !== "" && <Caption text={video.caption}/>}
            <div className="video-buttons">
                <a className="button button-primary" href={video.downloadUrl}>Download</a>
                <ConfirmAction
                    label="Remove"
                    question={`Remove ${video.name}?`}
                    run={() => clips.removeVideo({videoId: video.videoId})}
                    said={() => "Removed."}
                    refusal={removeRefusalOf}
                    onDone={onRemoved}
                    tone="danger"
                />
            </div>
        </li>
    )
}

function Caption({text}: {text: string}) {
    const box = useRef<HTMLParagraphElement>(null)
    const [copied, setCopied] = useState<boolean>()

    useEffect(() => {
        if (copied !== true) return
        const timer = setTimeout(() => setCopied(undefined), COPIED_MS)
        return () => clearTimeout(timer)
    }, [copied])

    const copy = () => copyText(text).then(done => {
        if (!done && box.current !== null) window.getSelection()?.selectAllChildren(box.current)
        setCopied(done)
    })

    return (
        <div className="video-caption">
            <p ref={box} className="video-caption-text">{text}</p>
            <button type="button" className="button" onClick={copy}>{copied === true ? "Copied" : "Copy caption"}</button>
            {copied === false && (
                <p className="action-refused" role="alert">The browser refused to copy. The caption is selected: copy it by hand.</p>
            )}
        </div>
    )
}
