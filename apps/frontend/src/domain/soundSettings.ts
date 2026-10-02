/**
 * Which sounds the player wants to hear. Stored in the browser, so it is read
 * back from whatever an older build — or a hand-edited localStorage — left
 * there, and anything it does not recognise falls back to the default.
 */

export const SOUNDS = [
    "click", "refused", "bonusSpawn", "bonusCaught", "spread", "enclose", "bomb", "chat",
    "quiz", "quizRight", "quizWrong",
] as const

export type SoundName = typeof SOUNDS[number]

/**
 * The sounds with a switch of their own. Every one of these is also a `SoundName`, which is what
 * lets the panel preview a switch by playing it.
 */
export const SWITCHES = ["click", "refused", "bonusSpawn", "bonusCaught", "bomb", "chat", "quiz"] as const

export type SwitchName = typeof SWITCHES[number]

/**
 * The switch a sound answers to. A sound is its own switch unless it is listed here.
 *
 * A bonus click is a click, so the click switch covers it. **A quiz is one switch for all three
 * of its sounds** — the banner, the right answer and the wrong one are one feature happening once,
 * and three lines in the panel for a thing that makes three noises in ten seconds is three lines
 * nobody wants. `quiz` is the banner, so it is also what the panel plays as the preview.
 */
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
    /** The master switch. Off, nothing plays whatever the rest say. */
    enabled: boolean
    sounds: Record<SwitchName, boolean>
    /** The leader's national anthem, under the game. `volume` is 0…1. */
    anthem: {on: boolean, volume: number}
}

export const SOUND_SETTINGS_STORAGE_KEY = "clickplanet-sound-settings"

export const DEFAULT_SOUND_SETTINGS: SoundSettings = {
    enabled: true,
    sounds: {click: true, refused: true, bonusSpawn: true, bonusCaught: true, bomb: true, chat: true, quiz: true},
    anthem: {on: true, volume: 0.3},
}

/** The master switch off. Nothing plays, whatever the rest of the settings say. */
export const MUTED_SOUND_SETTINGS: SoundSettings = {...DEFAULT_SOUND_SETTINGS, enabled: false}

/**
 * `whenUnset` is what nothing saved — or something unreadable — means. The dev
 * server passes the muted settings there, so a page opened while working starts
 * silent; a field the saved settings are missing still falls back to the
 * default, because it says what that field means, not whether sound is wanted.
 */
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
