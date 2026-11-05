import {quizShown, useQuiz} from '../quiz/quizStore.ts'
import './ConnectionLost.css'

export default function ConnectionLost() {
    const lowered = useQuiz(quizShown)
    return <div className={`connection-lost${lowered ? " connection-lost--lowered" : ""}`} role="alert">
        <div className="connection-lost-line panel">
            <span aria-hidden="true">📡</span>
            <span><strong>Connection lost</strong> Check your network</span>
        </div>
    </div>
}
