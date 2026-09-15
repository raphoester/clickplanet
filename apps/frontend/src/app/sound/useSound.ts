import {useCallback, useEffect, useRef, useState} from "react";
import {
    isAudible,
    parseSoundSettings,
    SOUND_SETTINGS_STORAGE_KEY,
    SoundName,
    SoundSettings,
} from "../../domain/soundSettings.ts";
import {createSoundPlayer, PlaySound, SoundPlayer} from "./soundPlayer.ts";

function readStoredSettings(): string | null {
    try {
        return window.localStorage.getItem(SOUND_SETTINGS_STORAGE_KEY)
    } catch {
        return null
    }
}

/**
 * The settings and the one player for the page. `play` never changes identity,
 * and reads the settings through a ref: the globe holds it for its whole life,
 * and a toggle must not rebuild the globe.
 */
export function useSound() {
    const [settings, setSettings] = useState<SoundSettings>(() => parseSoundSettings(readStoredSettings()))

    const settingsRef = useRef(settings)
    settingsRef.current = settings

    const playerRef = useRef<SoundPlayer | null>(null)
    if (!playerRef.current) {
        playerRef.current = createSoundPlayer({isAudible: (name) => isAudible(settingsRef.current, name)})
    }

    useEffect(() => {
        try {
            window.localStorage.setItem(SOUND_SETTINGS_STORAGE_KEY, JSON.stringify(settings))
        } catch (e) {
            console.error("Could not persist the sound settings", e)
        }
    }, [settings])

    // Browsers keep audio muted until the page is touched. Capture, so a
    // listener that stops propagation cannot keep it locked.
    useEffect(() => {
        const unlock = () => playerRef.current?.unlock()
        const options = {capture: true}
        window.addEventListener("pointerdown", unlock, options)
        window.addEventListener("keydown", unlock, options)
        return () => {
            window.removeEventListener("pointerdown", unlock, options)
            window.removeEventListener("keydown", unlock, options)
        }
    }, [])

    // Closes the context only: the player itself stays usable and opens a new
    // one on the next gesture, which StrictMode's unmount and remount rely on.
    useEffect(() => () => playerRef.current?.dispose(), [])

    const play: PlaySound = useCallback((name, options) => playerRef.current?.play(name, options), [])
    const preview = useCallback((name: SoundName) => playerRef.current?.preview(name), [])

    return {settings, setSettings, play, preview}
}
