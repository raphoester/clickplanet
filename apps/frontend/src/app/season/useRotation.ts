import {useEffect, useState} from "react"

export const TURN_MS = 6000

export type RotationHolds = {
    onMouseEnter: () => void
    onMouseLeave: () => void
    onFocus: () => void
    onBlur: () => void
}

export function useRotation(count: number): {shown: number, holds: RotationHolds} {
    const [turns, setTurns] = useState(0)
    const [hovered, setHovered] = useState(false)
    const [focused, setFocused] = useState(false)
    const held = hovered || focused

    useEffect(() => {
        if (count < 2 || held) return
        const timer = setInterval(() => setTurns((turns) => turns + 1), TURN_MS)
        return () => clearInterval(timer)
    }, [count, held])

    return {
        shown: count > 0 ? turns % count : 0,
        holds: {
            onMouseEnter: () => setHovered(true),
            onMouseLeave: () => setHovered(false),
            onFocus: () => setFocused(true),
            onBlur: () => setFocused(false),
        },
    }
}
