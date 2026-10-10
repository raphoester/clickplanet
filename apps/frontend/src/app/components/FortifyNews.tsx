import {useEffect, useRef} from 'react'
import {Countries} from '../../domain/countries.ts'
import {Fortified} from '../../domain/fortify.ts'
import CountryFlag from './CountryFlag.tsx'
import BonusIcon from './BonusIcon.tsx'
import './BombNews.css'
import './FortifyNews.css'

export const FORTIFY_NEWS_MS = 4000

export type FortifyNewsProps = {
    fortified: Fortified
    lowered?: boolean
    onDone: () => void
}

export default function FortifyNews({fortified, lowered, onDone}: FortifyNewsProps) {
    const {fortification, name, tiles} = fortified
    const country = Countries.get(fortification.countryId)?.name ?? fortification.countryId

    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), FORTIFY_NEWS_MS)
        return () => clearTimeout(timer)
    }, [fortified])

    return <div className={`bomb-news fortify-news${lowered ? " bomb-news--lowered" : ""}`} role="status" aria-live="polite">
        <div className="bomb-news-line fortify-news-line panel">
            <span className="fortify-news-shield"><BonusIcon kind="shields"/></span>
            <CountryFlag code={fortification.countryId}/>
            <span><strong>{country}</strong> fortified <strong>{name}</strong></span>
            <span className="fortify-news-tiles">+1 shield on {tiles.toLocaleString("en-US")} {tiles === 1 ? "tile" : "tiles"}</span>
        </div>
    </div>
}
