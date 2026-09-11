import {useCallback, useEffect, useRef, useState} from "react";
import {CapturedFrame} from "../viewer/capture.ts";
import {ShareStats} from "../../domain/shareCard.ts";
import {takePicture} from "./takePicture.ts";

/**
 * What the camera button produced: a picture to look at and send, or the news
 * that it could not be drawn. Both open the preview — a button that does
 * nothing visible when it fails is a button the player presses again.
 */
export type Shot =
    | {kind: "ready", file: File, /** For the preview `img`; revoked on discard. */ url: string}
    | {kind: "failed"}

/**
 * One picture at a time, and the object URL that shows it.
 *
 * The URL is the only thing here that has to be cleaned up by hand: it pins the
 * whole PNG in memory until it is revoked, and these run to about a megabyte.
 */
export function useSharePicture(
    capture: (() => Promise<CapturedFrame>) | undefined,
    stats: ShareStats,
) {
    const [shot, setShot] = useState<Shot>()
    const [taking, setTaking] = useState(false)
    const shown = useRef<string>()

    const discard = useCallback(() => {
        if (shown.current) URL.revokeObjectURL(shown.current)
        shown.current = undefined
        setShot(undefined)
    }, [])

    // Closing the menu, or leaving the page, is a discard like any other.
    useEffect(() => discard, [discard])

    const take = async () => {
        // Pressing it twice over would capture two frames and leak the first
        // one's URL, and the button is disabled while this is true anyway.
        if (!capture || taking) return

        setTaking(true)
        try {
            const file = await takePicture(capture, stats)
            shown.current = URL.createObjectURL(file)
            setShot({kind: "ready", file, url: shown.current})
        } catch (error) {
            console.error("Failed to take a picture of the globe", error)
            setShot({kind: "failed"})
        } finally {
            setTaking(false)
        }
    }

    return {shot, taking, take, discard}
}
