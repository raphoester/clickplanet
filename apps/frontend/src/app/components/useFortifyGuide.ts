import {useCallback, useState} from 'react'
import {FORTIFY_GUIDE_STORAGE_KEY} from '../../domain/fortify.ts'

function readSeen(): boolean {
    try {
        return window.localStorage.getItem(FORTIFY_GUIDE_STORAGE_KEY) === "seen"
    } catch {
        return false
    }
}

export function useFortifyGuide() {
    const [seen, setSeen] = useState(readSeen)

    const markSeen = useCallback(() => {
        setSeen(true)
        try {
            window.localStorage.setItem(FORTIFY_GUIDE_STORAGE_KEY, "seen")
        } catch (e) {
            console.error("Could not persist the fortify guide", e)
        }
    }, [])

    return {seen, markSeen}
}
