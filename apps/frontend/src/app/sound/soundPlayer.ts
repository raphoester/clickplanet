import {SoundName} from "../../domain/soundSettings.ts";
import {Synth, SynthOptions, SYNTHS} from "./synths.ts";

export type PlaySound = (name: SoundName, options?: Partial<SynthOptions>) => void

/**
 * The shortest gap between two plays of one sound, in milliseconds. A player
 * clicking fast, a burst of refusals or a busy chat would otherwise be one long
 * buzz — and every play is a handful of audio nodes.
 */
export const MIN_GAP_MS: Record<SoundName, number> = {
    click: 35,
    refused: 250,
    bonusSpawn: 0,
    bonusCaught: 0,
    spread: 60,
    boost: 35,
    enclose: 200,
    bomb: 150,
    chat: 1500,
}

/**
 * How long a sound asked for while the context was still waking up may start
 * late. Past it the moment it belonged to is gone, and it is dropped.
 */
const LATE_MS = 300

export type SoundPlayerOptions = {
    isAudible: (name: SoundName) => boolean
    createContext?: () => AudioContext | undefined
    synths?: Record<SoundName, Synth>
    now?: () => number
    isHidden?: () => boolean
}

export type SoundPlayer = {
    /** A sound from the game, if the settings allow it and it is not too soon. */
    play: PlaySound
    /** A sound the player just switched on, so they hear what they chose. */
    preview: (name: SoundName) => void
    /** Call from inside a user gesture: browsers keep audio muted until one. */
    unlock: () => void
    /** Closes the audio context. The next `unlock` opens a new one. */
    dispose: () => void
}

export function createSoundPlayer(options: SoundPlayerOptions): SoundPlayer {
    const {
        isAudible,
        createContext = defaultContext,
        synths = SYNTHS,
        now = () => performance.now(),
        isHidden = () => document.hidden,
    } = options

    let ctx: AudioContext | undefined
    const lastPlayed = new Map<SoundName, number>()

    const start = (name: SoundName, options: SynthOptions) => {
        if (!ctx) return
        const context = ctx

        if (context.state === "running") {
            synths[name](context, context.currentTime, options)
            return
        }

        // The context is only ever created by a gesture, so a suspended one is
        // one that gesture is still resuming — or one the browser took back.
        const asked = now()
        context.resume()
            .then(() => {
                if (context.state === "running" && now() - asked <= LATE_MS) {
                    synths[name](context, context.currentTime, options)
                }
            })
            .catch(() => {
            })
    }

    return {
        play: (name, {volume = 1, onWater = false} = {}) => {
            if (!ctx || !isAudible(name) || isHidden()) return

            const t = now()
            const last = lastPlayed.get(name)
            if (last !== undefined && t - last < MIN_GAP_MS[name]) return
            lastPlayed.set(name, t)

            start(name, {volume, onWater})
        },
        preview: (name) => {
            if (!ctx) ctx = createContext()
            start(name, {volume: 1, onWater: false})
        },
        unlock: () => {
            if (!ctx) ctx = createContext()
            if (ctx && ctx.state !== "running") ctx.resume().catch(() => {
            })
        },
        dispose: () => {
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
