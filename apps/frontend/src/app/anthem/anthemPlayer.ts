/**
 * Plays one anthem at a time on a loop, and crossfades when it changes.
 *
 * Each recording streams through an `<audio>` element, but its loudness goes
 * through Web Audio gain nodes: iOS ignores `HTMLMediaElement.volume`, so a fade
 * or a volume slider written to the element does nothing on an iPhone. The files
 * are served from our own origin, so routing them through the context does not
 * silence them the way a cross-origin source would.
 */

export const FADE_S = 2

export type AnthemPlayer = {
    /** The recording to play, or undefined for silence. Crossfades from the last. */
    setTrack: (url: string | undefined) => void
    /** Whether anything should be heard: the settings, and the tab being visible. */
    setAudible: (audible: boolean) => void
    /** 0…1. */
    setVolume: (volume: number) => void
    /** Call from inside a user gesture: browsers keep audio muted until one. */
    unlock: () => void
    /** Where the current recording is, for the player's progress bar. */
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
        const source = context.createMediaElementSource(audio)
        const gain = context.createGain()
        gain.gain.setValueAtTime(0, context.currentTime)
        gain.gain.linearRampToValueAtTime(1, context.currentTime + FADE_S)
        source.connect(gain).connect(out)
        if (audible) audio.play().catch(() => {
        })
        return {url, audio, gain, source}
    }

    // Brings what plays in line with what is wanted. Before the first gesture
    // there is no context, and this only remembers.
    const sync = () => {
        if (!ctx || !master) return
        if (current?.url === wantedUrl) return

        if (current) retire(current, ctx)
        current = wantedUrl ? start(wantedUrl, ctx, master) : undefined
    }

    const applyLevel = () => {
        if (!ctx || !master) return
        const now = ctx.currentTime
        master.gain.cancelScheduledValues(now)
        master.gain.setTargetAtTime(level(), now, 0.25)

        const audio = current?.audio
        if (!audio) return
        if (audible && audio.paused) audio.play().catch(() => {
        })
        // Paused rather than left running silent, so a muted anthem does not
        // keep streaming — once the fade has had time to finish.
        if (!audible && !audio.paused) {
            setTimeout(() => {
                if (!audible && current?.audio === audio) audio.pause()
            }, 1000)
        }
    }

    return {
        setTrack: (url) => {
            wantedUrl = url
            sync()
        },
        setAudible: (value) => {
            if (value === audible) return
            audible = value
            applyLevel()
        },
        setVolume: (value) => {
            volume = value
            applyLevel()
        },
        unlock: () => {
            if (!ctx) {
                ctx = createContext()
                if (!ctx) return
                master = ctx.createGain()
                master.gain.value = level()
                master.connect(ctx.destination)
                sync()
            }
            if (ctx.state !== "running") ctx.resume().catch(() => {
            })
            if (audible && current?.audio.paused) current.audio.play().catch(() => {
            })
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
