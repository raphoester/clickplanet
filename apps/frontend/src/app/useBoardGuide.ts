import {useCallback, useState} from "react"

export const BOARD_GUIDE_STORAGE_KEY = "clickplanet-board-guide"

function readGuided(): boolean {
    try {
        return window.localStorage.getItem(BOARD_GUIDE_STORAGE_KEY) !== null
    } catch {
        return false
    }
}

export function useBoardGuide() {
    const [guided, setGuided] = useState(readGuided)

    const markGuided = useCallback(() => {
        setGuided(true)
        try {
            window.localStorage.setItem(BOARD_GUIDE_STORAGE_KEY, "points")
        } catch (e) {
            console.error("Could not persist the board guide", e)
        }
    }, [])

    return {guided, markGuided}
}
