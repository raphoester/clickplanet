import {useEffect, useRef} from 'react'
import {BombDrop} from '../../backends/backend.ts'
import {Countries} from '../../domain/countries.ts'
import CountryFlag from './CountryFlag.tsx'
import {describeBlast} from '../../domain/blast.ts'
import './BombNews.css'

/** How long the line stays up. Matched to the `bomb-news` keyframes. */
export const BOMB_NEWS_MS = 4000

export type BombNewsProps = {
    drop: BombDrop
    onDone: () => void
}

/**
 * One line at the top of the screen when a bomb lands anywhere on the planet.
 * Most blasts happen where the player is not looking, so this is how they hear
 * about them at all. Nothing here is clickable.
 */
export default function BombNews({drop, onDone}: BombNewsProps) {
    const name = Countries.get(drop.countryId)?.name ?? drop.countryId

    // Held in a ref so only a new drop restarts the countdown; see BonusAward.
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), BOMB_NEWS_MS)
        return () => clearTimeout(timer)
    }, [drop])

    return <div className="bomb-news" role="status" aria-live="polite">
        <div className="bomb-news-line">
            <span aria-hidden="true">{drop.tile === undefined ? "🌊" : "💥"}</span>
            <CountryFlag code={drop.countryId}/>
            <span><strong>{name}</strong> {describeBlast(drop)}</span>
        </div>
    </div>
}
