export const FADE_S = 2

// A fade out under setTargetAtTime's 0.25 s constant is all but silent by then.
const QUIET_MS = 1000

export type AnthemPlayer = {
    setTrack: (url: string | undefined) => void
    setAudible: (audible: boolean) => void
    setVolume: (volume: number) => void
    unlock: () => void
    position: () => {current: number, duration: number} | undefined
    dispose: () => void
}

export type AnthemPlayerOptions = {
    createContext?: () => AudioContext | undefined
    createAudio?: (url: string) => HTMLAudioElement
}

type Track = {
    url: string
    audio: HTMLAudioElement
    gain: GainNode
    source: MediaElementAudioSourceNode
}

export function createAnthemPlayer(options: AnthemPlayerOptions = {}): AnthemPlayer {
    const {createContext = defaultContext, createAudio = defaultAudio} = options

    let ctx: AudioContext | undefined
    let master: GainNode | undefined
    let current: Track | undefined
    let wantedUrl: string | undefined
    let audible = false
    let volume = 0
    let unlocked = false

    const level = () => (audible ? volume : 0)

    const retire = (track: Track, context: AudioContext) => {
        const end = context.currentTime + FADE_S
        track.gain.gain.cancelScheduledValues(context.currentTime)
        track.gain.gain.setValueAtTime(track.gain.gain.value, context.currentTime)
        track.gain.gain.linearRampToValueAtTime(0, end)
        setTimeout(() => {
            track.audio.pause()
            track.source.disconnect()
            track.gain.disconnect()
            track.audio.removeAttribute("src")
            track.audio.load()
        }, FADE_S * 1000 + 100)
    }

    const start = (url: string, context: AudioContext, out: GainNode): Track => {
        const audio = createAudio(url)
        audio.loop = true
        // Loudness goes through Web Audio: iOS ignores HTMLMediaElement.volume.
        const source = context.createMediaElementSource(audio)
        const gain = context.createGain()
        gain.gain.setValueAtTime(0, context.currentTime)
        gain.gain.linearRampToValueAtTime(1, context.currentTime + FADE_S)
        source.connect(gain).connect(out)
        if (audible) audio.play().catch(() => {
        })
        return {url, audio, gain, source}
    }

    // A recording is fetched only to be heard.
    const sync = () => {
        if (!ctx || !master || !audible) return
        if (current?.url === wantedUrl) return

        if (current) retire(current, ctx)
        current = wantedUrl ? start(wantedUrl, ctx, master) : undefined
    }

    // Made only once a gesture has been seen and the anthem is heard: a running context keeps the audio hardware awake.
    const open = () => {
        if (ctx || !unlocked || !audible) return
        ctx = createContext()
        if (!ctx) return
        master = ctx.createGain()
        master.gain.value = level()
        master.connect(ctx.destination)
    }

    const applyLevel = () => {
        if (!ctx || !master) return
        const context = ctx
        const now = context.currentTime
        master.gain.cancelScheduledValues(now)
        master.gain.setTargetAtTime(level(), now, 0.25)

        if (audible) {
            if (context.state !== "running") context.resume().catch(() => {
            })
            sync()
            if (current?.audio.paused) current.audio.play().catch(() => {
            })
            return
        }

        const audio = current?.audio
        setTimeout(() => {
            if (audible || ctx !== context) return
            if (audio && current?.audio === audio) audio.pause()
            context.suspend().catch(() => {
            })
        }, QUIET_MS)
    }

    return {
        setTrack: (url) => {
            wantedUrl = url
            sync()
        },
        setAudible: (value) => {
            if (value === audible) return
            audible = value
            open()
            applyLevel()
        },
        setVolume: (value) => {
            volume = value
            applyLevel()
        },
        unlock: () => {
            unlocked = true
            if (!audible) return
            open()
            applyLevel()
        },
        position: () => {
            const audio = current?.audio
            if (!audio || !Number.isFinite(audio.duration)) return undefined
            return {current: audio.currentTime, duration: audio.duration}
        },
        dispose: () => {
            current?.audio.pause()
            current = undefined
            master = undefined
            ctx?.close().catch(() => {
            })
            ctx = undefined
        },
    }
}

function defaultContext(): AudioContext | undefined {
    if (typeof AudioContext === "undefined") return undefined
    try {
        return new AudioContext()
    } catch {
        return undefined
    }
}

function defaultAudio(url: string): HTMLAudioElement {
    const audio = new Audio()
    audio.preload = "auto"
    audio.src = url
    return audio
}
