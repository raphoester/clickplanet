import {useCallback, useEffect, useRef, useState} from "react";
import {CapturedFrame} from "../viewer/capture.ts";
import {ShareStats} from "../../domain/shareCard.ts";
import {takePicture} from "./takePicture.ts";

export type Shot =
    | {kind: "ready", file: File, url: string}
    | {kind: "failed"}

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

    useEffect(() => discard, [discard])

    const take = async () => {
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
