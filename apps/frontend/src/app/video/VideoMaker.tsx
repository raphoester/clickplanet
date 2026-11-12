import {useEffect, useState} from "react"
import {ShownNamesBackend} from "../../backends/clips.ts"
import {RecentHistorySource} from "../../backends/recentHistory.ts"
import {shareAddress, shareLabel, shareUrl} from "../../domain/shareCard.ts"
import {alliesOf, castOf, headlineOf, RecentHistory, storyOf} from "../../domain/video/story.ts"
import {MapHistory} from "../../domain/mapHistory.ts"
import {nameOf, said} from "../../clip/names.ts"
import {VideoJob} from "../../video/job.ts"
import Modal from "../components/Modal.tsx"
import ShareActions from "../share/ShareActions.tsx"
import {videoDeliveriesOffered} from "../share/deliverShare.ts"
import {makeVideo} from "./makeVideo.ts"
import {readHistory, stale, useVideo} from "./videoStore.ts"
import "./VideoMaker.css"

// The allies asked about: enough that three are left once the hidden names are out.
const MOST_ALLIES_ASKED = 30

type Making =
    | {kind: "reading"}
    | {kind: "making", share: number}
    | {kind: "ready", file: File, url: string, text: string}
    | {kind: "empty"}
    | {kind: "failed"}

export type VideoMakerProps = {
    source: RecentHistorySource
    names: ShownNamesBackend
    flag: string
    accountId: string | undefined
    username: string | undefined
    onSignIn?: () => void
    onClose: () => void
}

// The page carries no account to the video: only what it draws.
function playedOf(history: RecentHistory): MapHistory {
    return {frames: history.frames.map(({at, tiles, owners, changedAt}) => ({at, tiles, owners, changedAt})), hits: history.hits}
}

export default function VideoMaker({source, names, flag, accountId, username, onSignIn, onClose}: VideoMakerProps) {
    const [making, setMaking] = useState<Making>({kind: "reading"})

    useEffect(() => {
        const controller = new AbortController()
        const {signal} = controller
        const set = (next: Making) => {
            if (!signal.aborted) setMaking(next)
        }
        let url: string | undefined
        void (async () => {
            const kept = useVideo.getState().read
            const read = kept === undefined || stale(kept, Date.now()) ? await readHistory(source) : kept
            const story = storyOf(read.history, read.ground, flag)
            if (story === undefined) {
                set({kind: "empty"})
                return
            }
            const allies = alliesOf(read.history, read.ground, story, accountId).slice(0, MOST_ALLIES_ASKED)
            const shown = await names.shownNames(accountId === undefined ? allies : [accountId, ...allies]).catch((e) => {
                console.error("could not read which names a video may show", e)
                return new Map<string, string>()
            })
            const cast = castOf(accountId, allies, shown)
            const address = shareAddress(flag, username, accountId === undefined ? undefined : shown.get(accountId))
            const headline = headlineOf(story, {nameOf, said})
            const job: VideoJob = {history: playedOf(read.history), story, headline, cast, address: shareLabel(address)}
            set({kind: "making", share: 0})
            const {video} = await makeVideo(job, (share) => set({kind: "making", share}), signal)
            const file = new File([video], `clickplanet-${flag}.mp4`, {type: "video/mp4"})
            url = URL.createObjectURL(file)
            set({kind: "ready", file, url, text: `${headline} ${shareUrl(address)}`})
        })().catch((e) => {
            if (signal.aborted) return
            console.error("could not make the video", e)
            set({kind: "failed"})
        })
        return () => {
            controller.abort()
            if (url !== undefined) URL.revokeObjectURL(url)
        }
    }, [source, names, flag, accountId, username])

    return <Modal title="Your video"
                  className="modal-video"
                  footer={making.kind === "ready"
                      ? <ShareActions file={making.file} text={making.text} offered={videoDeliveriesOffered}/>
                      : undefined}
                  onClose={onClose}>
        <Body making={making} onSignIn={onSignIn && (() => {
            onClose()
            onSignIn()
        })}/>
    </Modal>
}

function Body({making, onSignIn}: {making: Making, onSignIn?: () => void}) {
    switch (making.kind) {
        case "reading":
        case "making": {
            const percent = making.kind === "making" ? Math.round(making.share * 100) : 0
            return <div className="video-making">
                <p>Making your video…</p>
                <div className="video-progress" role="progressbar" aria-label="Video made" aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}>
                    <div className="video-progress-fill" style={{width: `${percent}%`}}/>
                </div>
            </div>
        }
        case "ready":
            return <div className="video-ready">
                <video className="video-preview" src={making.url} controls playsInline loop autoPlay muted/>
                {onSignIn && <button type="button" className="button button-action" onClick={onSignIn}>Sign in to put your name on it</button>}
            </div>
        case "empty":
            return <p className="video-failed">Take some land first.</p>
        case "failed":
            return <p className="video-failed">The video could not be made.</p>
    }
}
