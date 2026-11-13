import {ReactNode, useEffect, useRef} from 'react'
import {BombDrop} from '../../backends/backend.ts'
import CountryFlag from '../components/CountryFlag.tsx'
import {Blast, blastOf} from '../../domain/blast.ts'
import {translator} from '../../i18n/page.ts'
import {rich} from '../../i18n/rich.tsx'
import './BombNews.css'

const BOMB_NEWS_MS = 4000

export type BombNewsProps = {
    drop: BombDrop
    land?: string

    onDone: () => void
}

export default function BombNews({drop, land, onDone}: BombNewsProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), BOMB_NEWS_MS)
        return () => clearTimeout(timer)
    }, [drop])

    return <div className="bomb-news" role="status" aria-live="polite">
        <div className="bomb-news-line panel">
            <span aria-hidden="true">{drop.tile === undefined ? "🌊" : "💥"}</span>
            <CountryFlag code={drop.countryId}/>
            <span>{bombLine(blastOf({tile: drop.tile, cleared: drop.cleared.length}, land), <strong>{translator().country(drop.countryId)}</strong>)}</span>
        </div>
    </div>
}

function bombLine(blast: Blast, flag: ReactNode): ReactNode {
    switch (blast.kind) {
        case "ocean":
            return rich("news.bomb.ocean", {flag})
        case "land":
            return rich("news.bomb.land", {flag, land: blast.land})
        case "empty":
            return rich("news.bomb.empty", {flag})
        case "tiles":
            return rich("news.bomb.tiles", {flag, count: blast.count})
    }
}
