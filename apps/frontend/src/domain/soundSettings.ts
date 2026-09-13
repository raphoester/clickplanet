/**
 * Which sounds the player wants to hear. Stored in the browser, so it is read
 * back from whatever an older build — or a hand-edited localStorage — left
 * there, and anything it does not recognise falls back to the default.
 */

export const SOUNDS = ["click", "refused", "bonusSpawn", "bonusCaught", "spread", "boost", "enclose", "bomb", "chat"] as const

export type SoundName = typeof SOUNDS[number]

/** The sounds with a switch of their own. */
export const SWITCHES = ["click", "refused", "bonusSpawn", "bonusCaught", "bomb", "chat"] as const

export type SwitchName = typeof SWITCHES[number]

/** A bonus click is a click, so the click switch covers it. */
export function switchOf(name: SoundName): SwitchName {
    switch (name) {
        case "spread":
        case "boost":
        case "enclose":
            return "click"
        default:
            return name
    }
}

export type SoundSettings = {
    /** The master switch. Off, nothing plays whatever the rest say. */
    enabled: boolean
    sounds: Record<SwitchName, boolean>
    /** The leader's national anthem, under the game. `volume` is 0…1. */
    anthem: {on: boolean, volume: number}
}

export const SOUND_SETTINGS_STORAGE_KEY = "clickplanet-sound-settings"

export const DEFAULT_SOUND_SETTINGS: SoundSettings = {
    enabled: true,
    sounds: {click: true, refused: true, bonusSpawn: true, bonusCaught: true, bomb: true, chat: true},
    anthem: {on: true, volume: 0.3},
}

export function parseSoundSettings(raw: string | null): SoundSettings {
    if (raw === null) return DEFAULT_SOUND_SETTINGS

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return DEFAULT_SOUND_SETTINGS
    }
    if (typeof stored !== "object" || stored === null) return DEFAULT_SOUND_SETTINGS

    const {enabled, sounds, anthem} = stored as {enabled?: unknown, sounds?: unknown, anthem?: unknown}
    const storedSounds = typeof sounds === "object" && sounds !== null ? sounds as Record<string, unknown> : {}

    // A sound this build added after the settings were saved starts on, like
    // every other sound does.
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
