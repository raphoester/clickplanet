import {useCallback, useEffect, useRef, useState} from "react";
import {
    DEFAULT_SOUND_SETTINGS,
    isAudible,
    MUTED_SOUND_SETTINGS,
    parseSoundSettings,
    SOUND_SETTINGS_STORAGE_KEY,
    SoundName,
    SoundSettings,
} from "../../domain/soundSettings.ts";
import {createSoundPlayer, PlaySound, SoundPlayer} from "./soundPlayer.ts";

const STORAGE_KEY = import.meta.env.DEV ? `${SOUND_SETTINGS_STORAGE_KEY}-dev` : SOUND_SETTINGS_STORAGE_KEY
const WHEN_UNSET = import.meta.env.DEV ? MUTED_SOUND_SETTINGS : DEFAULT_SOUND_SETTINGS

function readStoredSettings(): string | null {
    try {
        return window.localStorage.getItem(STORAGE_KEY)
    } catch {
        return null
    }
}

export function useSound() {
    const [settings, setSettings] = useState<SoundSettings>(() => parseSoundSettings(readStoredSettings(), WHEN_UNSET))

    const settingsRef = useRef(settings)
    settingsRef.current = settings

    const playerRef = useRef<SoundPlayer | null>(null)
    if (!playerRef.current) {
        playerRef.current = createSoundPlayer({isAudible: (name) => isAudible(settingsRef.current, name)})
    }

    useEffect(() => {
        try {
            window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
        } catch (e) {
            console.error("Could not persist the sound settings", e)
        }
    }, [settings])

    useEffect(() => {
        const unlock = () => playerRef.current?.unlock()
        // Capture, so a listener that stops propagation cannot keep the audio locked.
        const options = {capture: true}
        window.addEventListener("pointerdown", unlock, options)
        window.addEventListener("keydown", unlock, options)
        return () => {
            window.removeEventListener("pointerdown", unlock, options)
            window.removeEventListener("keydown", unlock, options)
        }
    }, [])

    useEffect(() => () => playerRef.current?.dispose(), [])

    const play: PlaySound = useCallback((name, options) => playerRef.current?.play(name, options), [])
    const preview = useCallback((name: SoundName) => playerRef.current?.preview(name), [])

    return {settings, setSettings, play, preview}
}
