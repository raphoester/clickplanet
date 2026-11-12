import {
    BufferSource,
    BufferTarget,
    EncodedAudioPacketSource,
    EncodedPacket,
    EncodedPacketSink,
    EncodedVideoPacketSource,
    Input,
    MP4,
    Mp4OutputFormat,
    Output,
} from "mediabunny"
import {VIDEO_FPS} from "./job.ts"

// Encoded frames waiting in the encoder, past which the drawing waits for it.
const MOST_QUEUED = 4

const KEY_FRAME_EVERY = 2 * VIDEO_FPS

// The anthem's AAC packets as they are in its file, for the video to carry as they are: no encoder of the browser's
// (iOS Safari's AAC encoder is broken).
export type Sound = {
    packets: EncodedPacket[]
    config: AudioDecoderConfig
}

export async function soundOf(url: string, from: number, seconds: number): Promise<Sound | undefined> {
    const answer = await fetch(url)
    if (!answer.ok) throw new Error(`${url} answered ${answer.status}`)
    const input = new Input({source: new BufferSource(await answer.arrayBuffer()), formats: [MP4]})
    const track = await input.getPrimaryAudioTrack()
    if (track === null || track.codec !== "aac") return undefined
    const config = await track.getDecoderConfig()
    if (config === null) return undefined
    const sink = new EncodedPacketSink(track)
    const first = await sink.getPacket(from) ?? await sink.getFirstPacket()
    if (first === null) return undefined

    const packets: EncodedPacket[] = []
    for await (const packet of sink.packets(first)) {
        const at = packet.timestamp - first.timestamp
        if (at >= seconds) break
        packets.push(packet.clone({timestamp: at}))
    }
    return {packets, config}
}

export type Recorder = {
    add(canvas: HTMLCanvasElement, frame: number): Promise<void>
    finish(): Promise<Blob>
}

// H.264 frames from the canvas as they are drawn, and the sound, in one MP4 with its index first.
export async function recorderOf(config: VideoEncoderConfig, sound: Sound | undefined): Promise<Recorder> {
    const output = new Output({format: new Mp4OutputFormat({fastStart: "in-memory"}), target: new BufferTarget()})
    const video = new EncodedVideoPacketSource("avc")
    output.addVideoTrack(video, {frameRate: VIDEO_FPS})
    const audio = sound && new EncodedAudioPacketSource("aac")
    if (audio) output.addAudioTrack(audio)
    await output.start()

    if (sound && audio) {
        for (const [i, packet] of sound.packets.entries()) await audio.add(packet, i === 0 ? {decoderConfig: sound.config} : undefined)
    }

    let failed: Error | undefined
    let muxed = Promise.resolve()
    const encoder = new VideoEncoder({
        output: (chunk, meta) => {
            muxed = muxed.then(() => video.add(EncodedPacket.fromEncodedChunk(chunk), meta))
        },
        error: (error) => {
            failed = error
        },
    })
    encoder.configure(config)

    const dequeued = () => new Promise<void>((resolve) => encoder.addEventListener("dequeue", () => resolve(), {once: true}))

    return {
        async add(canvas, frame) {
            if (failed) throw failed
            const picture = new VideoFrame(canvas, {timestamp: Math.round(frame * 1e6 / VIDEO_FPS), duration: Math.round(1e6 / VIDEO_FPS)})
            encoder.encode(picture, {keyFrame: frame % KEY_FRAME_EVERY === 0})
            picture.close()
            while (encoder.encodeQueueSize > MOST_QUEUED && !failed) await dequeued()
        },
        async finish() {
            await encoder.flush()
            encoder.close()
            if (failed) throw failed
            await muxed
            await output.finalize()
            const buffer = output.target.buffer
            if (buffer === null) throw new Error("the video was muxed into nothing")
            return new Blob([buffer], {type: "video/mp4"})
        },
    }
}
