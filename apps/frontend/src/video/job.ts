import {MapHistory} from "../domain/mapHistory.ts"
import {Cast, VideoStory} from "../domain/video/story.ts"

// What the game hands the video page: the 24 hours to play, the story and every word on screen.
export type VideoJob = {
    history: MapHistory
    story: VideoStory
    headline: string
    cast: Cast
    // The address as a person reads it: clickplanet.lol/pl/kutza.
    address: string
}

export type VideoReply =
    | {kind: "progress", share: number}
    | {kind: "done", video: Blob, silent: boolean}
    | {kind: "failed", message: string}

export const VIDEO_PAGE = "/video"

export const VIDEO_SIZE = {width: 720, height: 1280}

export const VIDEO_FPS = 30

// 6 s of the map, then the end card.
export const VIDEO_PLAY_SECONDS = 6

export const VIDEO_SECONDS = 8

// H.264 at level 3.1 holds 720x1280 at 30 fps: High first, then what a phone's encoder may only have.
const CODECS = ["avc1.64001f", "avc1.4d001f", "avc1.42001f"]

const BITRATE = 5_000_000

export function encoderConfigOf(codec: string): VideoEncoderConfig {
    return {
        codec,
        ...VIDEO_SIZE,
        bitrate: BITRATE,
        framerate: VIDEO_FPS,
        avc: {format: "avc"},
    }
}

// The first config this browser says it can encode, none where it cannot.
export async function encoderConfig(): Promise<VideoEncoderConfig | undefined> {
    if (typeof VideoEncoder === "undefined" || typeof VideoFrame === "undefined") return undefined
    for (const codec of CODECS) {
        const config = encoderConfigOf(codec)
        try {
            if ((await VideoEncoder.isConfigSupported(config)).supported) return config
        } catch {
            // a codec string the browser refuses to read is one it cannot encode
        }
    }
    return undefined
}
