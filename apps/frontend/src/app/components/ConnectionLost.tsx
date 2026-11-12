import {quizShown, useQuiz} from '../quiz/quizStore.ts'
import {rich} from '../../i18n/rich.tsx'
import './ConnectionLost.css'

export default function ConnectionLost() {
    const lowered = useQuiz(quizShown)
    return <div className={`connection-lost${lowered ? " connection-lost--lowered" : ""}`} role="alert">
        <div className="connection-lost-line panel">
            <span aria-hidden="true">📡</span>
            <span>{rich("connection.lost", {strong: (words) => <strong>{words}</strong>})}</span>
        </div>
    </div>
}
