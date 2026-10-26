import {useEffect, useRef} from "react"
import {BATTLE_OVER_MS} from "../../domain/battle.ts"
import {SwordsIcon} from "../components/icons.tsx"
import "./Battle.css"

export type BattleOverProps = {
    lowered?: boolean
    onDone: () => void
}

export default function BattleOver({lowered, onDone}: BattleOverProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), BATTLE_OVER_MS)
        return () => clearTimeout(timer)
    }, [])

    return <div className={`battle-over${lowered ? " battle-over--lowered" : ""}`} role="status" aria-live="polite">
        <div className="battle-over-line panel">
            <SwordsIcon/>
            <strong>The battle is over</strong>
        </div>
    </div>
}
