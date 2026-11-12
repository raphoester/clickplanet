import {useEffect, useRef} from 'react'
import {Fortified} from '../../domain/fortify.ts'
import CountryFlag from '../components/CountryFlag.tsx'
import BonusIcon from '../components/BonusIcon.tsx'
import {quizShown, useQuiz} from '../quiz/quizStore.ts'
import {t, translator} from '../../i18n/page.ts'
import {rich} from '../../i18n/rich.tsx'
import './BombNews.css'
import './FortifyNews.css'

export const FORTIFY_NEWS_MS = 4000

export type FortifyNewsProps = {
    fortified: Fortified
    onDone: () => void
}

export default function FortifyNews({fortified, onDone}: FortifyNewsProps) {
    const {fortification, name, tiles} = fortified
    const lowered = useQuiz(quizShown)

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
            <span>{rich("news.fortified", {flag: <strong>{translator().country(fortification.countryId)}</strong>, territory: <strong>{name}</strong>})}</span>
            <span className="fortify-news-tiles">{t("news.fortifyShields", {tiles})}</span>
        </div>
    </div>
}
