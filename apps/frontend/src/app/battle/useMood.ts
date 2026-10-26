import {useEffect} from "react"

export const MOOD_ATTRIBUTE = "data-mood"

export type Mood = "battle"

export function useMood(mood: Mood | undefined) {
    useEffect(() => {
        const root = document.documentElement
        if (mood) root.setAttribute(MOOD_ATTRIBUTE, mood)
        else root.removeAttribute(MOOD_ATTRIBUTE)
        return () => root.removeAttribute(MOOD_ATTRIBUTE)
    }, [mood])
}
