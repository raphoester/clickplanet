export const SOUNDS = [
    "click", "refused", "bonusSpawn", "bonusCaught", "spread", "enclose", "bomb", "chat",
    "quiz", "quizRight", "quizWrong", "title", "fortify",
] as const

export type SoundName = typeof SOUNDS[number]

export const SWITCHES = ["click", "refused", "bonusSpawn", "bonusCaught", "bomb", "fortify", "chat", "quiz", "title"] as const

export type SwitchName = typeof SWITCHES[number]

export function switchOf(name: SoundName): SwitchName {
    switch (name) {
        case "spread":
        case "enclose":
            return "click"
        case "quizRight":
        case "quizWrong":
            return "quiz"
        default:
            return name
    }
}

export type SoundSettings = {
    enabled: boolean
    sounds: Record<SwitchName, boolean>
    anthem: {on: boolean, volume: number}
}

export const SOUND_SETTINGS_STORAGE_KEY = "clickplanet-sound-settings"

export const DEFAULT_SOUND_SETTINGS: SoundSettings = {
    enabled: true,
    sounds: {click: true, refused: true, bonusSpawn: true, bonusCaught: true, bomb: true, fortify: true, chat: true, quiz: true, title: true},
    anthem: {on: true, volume: 0.3},
}

export const MUTED_SOUND_SETTINGS: SoundSettings = {...DEFAULT_SOUND_SETTINGS, enabled: false}

export function parseSoundSettings(raw: string | null, whenUnset: SoundSettings = DEFAULT_SOUND_SETTINGS): SoundSettings {
    if (raw === null) return whenUnset

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return whenUnset
    }
    if (typeof stored !== "object" || stored === null) return whenUnset

    const {enabled, sounds, anthem} = stored as {enabled?: unknown, sounds?: unknown, anthem?: unknown}
    const storedSounds = typeof sounds === "object" && sounds !== null ? sounds as Record<string, unknown> : {}

    const parsed = {} as Record<SwitchName, boolean>
    for (const name of SWITCHES) {
        const value = storedSounds[name]
        parsed[name] = typeof value === "boolean" ? value : DEFAULT_SOUND_SETTINGS.sounds[name]
    }

    return {
        enabled: typeof enabled === "boolean" ? enabled : DEFAULT_SOUND_SETTINGS.enabled,
        sounds: parsed,
        anthem: parseAnthem(anthem),
    }
}

function parseAnthem(stored: unknown): SoundSettings["anthem"] {
    const fallback = DEFAULT_SOUND_SETTINGS.anthem
    if (typeof stored !== "object" || stored === null) return fallback

    const {on, volume} = stored as {on?: unknown, volume?: unknown}
    return {
        on: typeof on === "boolean" ? on : fallback.on,
        volume: typeof volume === "number" && volume >= 0 && volume <= 1 ? volume : fallback.volume,
    }
}

export function isAnthemAudible(settings: SoundSettings): boolean {
    return settings.enabled && settings.anthem.on && settings.anthem.volume > 0
}

export function isAudible(settings: SoundSettings, name: SoundName): boolean {
    return settings.enabled && settings.sounds[switchOf(name)]
}
