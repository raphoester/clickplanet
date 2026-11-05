import {create} from "zustand"
import {
    DEFAULT_SOUND_SETTINGS,
    MUTED_SOUND_SETTINGS,
    parseSoundSettings,
    SOUND_SETTINGS_STORAGE_KEY,
    SoundSettings,
} from "../../domain/soundSettings.ts"

const STORAGE_KEY = import.meta.env.DEV ? `${SOUND_SETTINGS_STORAGE_KEY}-dev` : SOUND_SETTINGS_STORAGE_KEY
const WHEN_UNSET = import.meta.env.DEV ? MUTED_SOUND_SETTINGS : DEFAULT_SOUND_SETTINGS

type SoundSettingsStore = {
    settings: SoundSettings
    setSettings: (settings: SoundSettings) => void
}

function readStoredSettings(): string | null {
    try {
        return window.localStorage.getItem(STORAGE_KEY)
    } catch {
        return null
    }
}

export const useSoundSettings = create<SoundSettingsStore>()((set) => ({
    settings: parseSoundSettings(readStoredSettings(), WHEN_UNSET),
    setSettings: (settings) => set({settings}),
}))

useSoundSettings.subscribe(({settings}) => {
    try {
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
    } catch (e) {
        console.error("Could not persist the sound settings", e)
    }
})
