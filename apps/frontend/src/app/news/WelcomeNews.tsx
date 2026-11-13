import {useEffect, useRef} from "react"
import {t} from "../../i18n/page.ts"
import "./BombNews.css"
import "./WelcomeNews.css"

export const WELCOME_NEWS_MS = 4000

export type WelcomeNewsProps = {
    recruiter: string
    flagName: string
    onDone: () => void
}

export default function WelcomeNews({recruiter, flagName, onDone}: WelcomeNewsProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), WELCOME_NEWS_MS)
        return () => clearTimeout(timer)
    }, [recruiter, flagName])

    return <div className="bomb-news" role="status" aria-live="polite">
        <div className="bomb-news-line welcome-news-line panel">{t("news.welcome", {player: recruiter, flag: flagName})}</div>
    </div>
}
