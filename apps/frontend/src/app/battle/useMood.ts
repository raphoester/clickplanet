import {useEffect} from "react"
import {Mood} from "../../domain/mood.ts"

export const MOOD_ATTRIBUTE = "data-mood"

export function useMood(mood: Mood | undefined) {
    useEffect(() => {
        const root = document.documentElement
        if (mood) root.setAttribute(MOOD_ATTRIBUTE, mood)
        else root.removeAttribute(MOOD_ATTRIBUTE)
        return () => root.removeAttribute(MOOD_ATTRIBUTE)
    }, [mood])
}
