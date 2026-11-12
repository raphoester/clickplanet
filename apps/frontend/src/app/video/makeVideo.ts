import {VIDEO_PAGE, VIDEO_SIZE, VideoJob, VideoReply} from "../../video/job.ts"

export type MadeVideo = {video: Blob, silent: boolean}

// The video is drawn in a page of its own, out of sight: its clock is the video's and its globe is the replay's, and the
// game's own globe runs on.
export function makeVideo(job: VideoJob, onProgress: (share: number) => void, signal: AbortSignal): Promise<MadeVideo> {
    const frame = document.createElement("iframe")
    frame.src = `${VIDEO_PAGE}?gfx=aa,earth,compact`
    frame.setAttribute("aria-hidden", "true")
    frame.tabIndex = -1
    Object.assign(frame.style, {
        position: "fixed", left: "-10000px", top: "0", width: `${VIDEO_SIZE.width}px`, height: `${VIDEO_SIZE.height}px`,
        border: "0", opacity: "0", pointerEvents: "none",
    })

    return new Promise<MadeVideo>((resolve, reject) => {
        const channel = new MessageChannel()
        const end = () => {
            window.removeEventListener("message", onReady)
            signal.removeEventListener("abort", onAbort)
            channel.port1.close()
            frame.remove()
        }
        const onAbort = () => {
            end()
            reject(new DOMException("the video was let go", "AbortError"))
        }
        const onReady = (event: MessageEvent<{kind?: string}>) => {
            if (event.source !== frame.contentWindow || event.data?.kind !== "ready") return
            window.removeEventListener("message", onReady)
            frame.contentWindow?.postMessage({kind: "job", job}, location.origin, [channel.port2])
        }
        channel.port1.onmessage = ({data}: MessageEvent<VideoReply>) => {
            switch (data.kind) {
                case "progress":
                    onProgress(data.share)
                    return
                case "done":
                    end()
                    resolve({video: data.video, silent: data.silent})
                    return
                case "failed":
                    end()
                    reject(new Error(data.message))
                    return
            }
        }
        if (signal.aborted) {
            onAbort()
            return
        }
        signal.addEventListener("abort", onAbort)
        window.addEventListener("message", onReady)
        document.body.append(frame)
    })
}
