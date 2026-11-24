import {useEffect, useRef} from "react"
import {Decided} from "../../domain/flagSubmissions.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {rich} from "../../i18n/rich.tsx"
import {Key} from "../../i18n/translator.ts"
import "./BombNews.css"
import "./WelcomeNews.css"
import "../flags/FlagSubmission.css"

export const SUBMISSION_NEWS_MS = 4000

export type SubmissionNewsProps = {
    submission: Decided
    onDone: () => void
}

function lineOf({state, change}: Decided): Key {
    switch (state) {
        case "approved":
            return change ? "news.submission.changed" : "news.submission.approved"
        case "rejected":
            return "news.submission.rejected"
        case "opened":
            return "news.submission.opened"
    }
}

export default function SubmissionNews({submission, onDone}: SubmissionNewsProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), SUBMISSION_NEWS_MS)
        return () => clearTimeout(timer)
    }, [submission])

    return <div className="bomb-news" role="status" aria-live="polite">
        <div className="bomb-news-line welcome-news-line panel">
            {submission.state === "opened" && <CountryFlag code={submission.flag}/>}
            <span>{rich(lineOf(submission), {flag: <span className="submission-news-name">{submission.name}</span>})}</span>
        </div>
    </div>
}
