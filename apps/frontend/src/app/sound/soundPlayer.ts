import {SoundName} from "../../domain/soundSettings.ts";
import {Synth, SynthOptions, SYNTHS} from "./synths.ts";

export type PlaySound = (name: SoundName, options?: Partial<SynthOptions>) => void

export const MIN_GAP_MS: Record<SoundName, number> = {
    click: 35,
    refused: 250,
    bonusSpawn: 0,
    bonusCaught: 0,
    spread: 60,
    enclose: 200,
    bomb: 150,
    chat: 1500,
    quiz: 0,
    quizRight: 0,
    quizWrong: 0,
    title: 1000,
    fortify: 600,
}

const LATE_MS = 300

export type SoundPlayerOptions = {
    isAudible: (name: SoundName) => boolean
    createContext?: () => AudioContext | undefined
    synths?: Record<SoundName, Synth>
    now?: () => number
    isHidden?: () => boolean
}

export type SoundPlayer = {
    play: PlaySound
    preview: (name: SoundName) => void
    unlock: () => void
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
