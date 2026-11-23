import {useCallback, useEffect, useRef} from "react";
import {isAudible, SoundName} from "../../domain/soundSettings.ts";
import {createSoundPlayer, PlaySound, SoundPlayer} from "./soundPlayer.ts";
import {useSoundSettings} from "./soundSettingsStore.ts";

export function useSound() {
    const playerRef = useRef<SoundPlayer | null>(null)
    if (!playerRef.current) {
        playerRef.current = createSoundPlayer({
            isAudible: (name) => isAudible(useSoundSettings.getState().settings, name),
        })
    }

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

    return {play, preview}
}
