import {useEffect, useRef} from 'react'
import {CloseToFortify} from '../../domain/fortify.ts'
import BonusIcon from './BonusIcon.tsx'
import './BombNews.css'
import './FortifyNews.css'

export const FORTIFY_HINT_MS = 4000

export type FortifyHintProps = {
    close: CloseToFortify
    lowered?: boolean
    onDone: () => void
}

export default function FortifyHint({close, lowered, onDone}: FortifyHintProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), FORTIFY_HINT_MS)
        return () => clearTimeout(timer)
    }, [close])

    return <div className={`bomb-news fortify-news${lowered ? " bomb-news--lowered" : ""}`} role="status" aria-live="polite">
        <div className="bomb-news-line fortify-news-line panel">
            <span className="fortify-news-shield"><BonusIcon kind="shields"/></span>
            <span><strong>{close.name}</strong>: {close.missing} {close.missing === 1 ? "tile" : "tiles"} to fortify</span>
        </div>
    </div>
}
