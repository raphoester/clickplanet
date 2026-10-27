import {useEffect, useRef, useState} from "react"
import {BATTLE_OVER_MS} from "../../domain/battle.ts"
import {SwordsIcon} from "../components/icons.tsx"
import "./Battle.css"

export type BattleOverProps = {
    awaiting: boolean
    lowered?: boolean
    onDone: () => void
}

export default function BattleOver({awaiting, lowered, onDone}: BattleOverProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    // A line that waited for the brief goes as the brief comes.
    const [waited] = useState(awaiting)
    useEffect(() => {
        if (awaiting) return
        const timer = setTimeout(() => done.current(), waited ? 0 : BATTLE_OVER_MS)
        return () => clearTimeout(timer)
    }, [awaiting, waited])

    const className = ["battle-over", lowered && "battle-over--lowered", awaiting && "battle-over--awaiting"].filter(Boolean).join(" ")
    return <div className={className} role="status" aria-live="polite">
        <div className="battle-over-line panel">
            <SwordsIcon/>
            <span className="battle-over-text">
                <strong>The battle is over</strong>
                {awaiting && <span className="battle-over-coming">The results are coming</span>}
            </span>
        </div>
    </div>
}
