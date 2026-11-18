import {useCallback, useEffect, useState} from "react"
import type {BoardAsset} from "../../domain/board.ts"
import {WORLD_BEFORE, WORLD_GREW_STORAGE_KEY, type WorldGrew, worldGrewKept, worldGrewOn} from "../../domain/worldGrew.ts"
import {boardNamed} from "../viewer/boards.ts"

function seenBoard(): string | null {
    try {
        return window.localStorage.getItem(WORLD_GREW_STORAGE_KEY)
    } catch {
        return null
    }
}

function keepSeen(board: string) {
    try {
        window.localStorage.setItem(WORLD_GREW_STORAGE_KEY, board)
    } catch {
        // A browser with no storage is told again on its next page.
    }
}

// "The world grew", due once the board the page draws is known: see domain/worldGrew.ts.
export function useWorldGrew(board: Promise<BoardAsset>, firstVisit: boolean): {grew?: WorldGrew, dismiss: () => void} {
    const [grew, setGrew] = useState<WorldGrew>()

    useEffect(() => {
        let live = true
        board.then((drawn) => {
            const before = boardNamed(WORLD_BEFORE)
            if (!live || before === undefined) return
            const seen = seenBoard()
            if (worldGrewKept(drawn, seen, firstVisit)) keepSeen(drawn.name)
            setGrew(worldGrewOn(drawn, before, seen, firstVisit))
        }, () => {})
        return () => {
            live = false
        }
    }, [board, firstVisit])

    const dismiss = useCallback(() => {
        if (grew) keepSeen(grew.board)
        setGrew(undefined)
    }, [grew])

    return {grew, dismiss}
}
