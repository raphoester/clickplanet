import {create} from "zustand"

const BOARD_GUIDE_STORAGE_KEY = "clickplanet-board-guide"

type BoardGuideStore = {
    guided: boolean
    markGuided: () => void
}

function readGuided(): boolean {
    try {
        return window.localStorage.getItem(BOARD_GUIDE_STORAGE_KEY) !== null
    } catch {
        return false
    }
}

export const useBoardGuide = create<BoardGuideStore>()((set) => ({
    guided: readGuided(),
    markGuided: () => set({guided: true}),
}))

useBoardGuide.subscribe(({guided}) => {
    if (!guided) return
    try {
        window.localStorage.setItem(BOARD_GUIDE_STORAGE_KEY, "points")
    } catch (e) {
        console.error("Could not persist the board guide", e)
    }
})
