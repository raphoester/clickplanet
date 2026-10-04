import {useEffect, useRef} from 'react'
import {BombDrop} from '../../backends/backend.ts'
import {Countries} from '../../domain/countries.ts'
import CountryFlag from './CountryFlag.tsx'
import {describeBlast} from '../../domain/blast.ts'
import './BombNews.css'

export const BOMB_NEWS_MS = 4000

export type BombNewsProps = {
    drop: BombDrop
    land?: string

    lowered?: boolean

    onDone: () => void
}

export default function BombNews({drop, land, lowered, onDone}: BombNewsProps) {
    const name = Countries.get(drop.countryId)?.name ?? drop.countryId
    const landName = land === undefined ? undefined : Countries.get(land)?.name

    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), BOMB_NEWS_MS)
        return () => clearTimeout(timer)
    }, [drop])

    return <div className={`bomb-news${lowered ? " bomb-news--lowered" : ""}`} role="status" aria-live="polite">
        <div className="bomb-news-line panel">
            <span aria-hidden="true">{drop.tile === undefined ? "🌊" : "💥"}</span>
            <CountryFlag code={drop.countryId}/>
            <span><strong>{name}</strong> {describeBlast({tile: drop.tile, cleared: drop.cleared.length}, landName)}</span>
        </div>
    </div>
}
