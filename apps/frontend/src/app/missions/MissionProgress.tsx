import {ReactNode} from "react"
import {FlagMissionOrder, MissionOrder} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {Moved} from "../../domain/missionProgress.ts"
import {orderOf, reachedOf} from "../../domain/missions.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {quizShown, useQuiz} from "../quiz/quizStore.ts"
import "../news/BombNews.css"
import "./Missions.css"

export default function MissionProgress({moved}: {moved: Moved}) {
    const lowered = useQuiz(quizShown)
    const reached = reachedOf(moved.order)

    return <div className={`bomb-news mission-progress${lowered ? " bomb-news--lowered" : ""}`} role="status">
        <div className="bomb-news-line mission-progress-line panel">
            {moved.flag && <span className="mission-progress-kicker">Flag mission</span>}
            <span className="mission-progress-order">{lineOf(moved.order)}</span>
            <div className="mission-bar mission-progress-bar"
                 role="progressbar"
                 aria-valuemin={0}
                 aria-valuemax={moved.order.goal}
                 aria-valuenow={reached}
                 aria-label={orderOf(moved.order)}>
                <div className="mission-bar-filled" style={{width: `${reached / moved.order.goal * 100}%`}}/>
            </div>
            <span className="mission-progress-count">{reached}/{moved.order.goal}</span>
        </div>
    </div>
}

function lineOf(order: MissionOrder | FlagMissionOrder): ReactNode {
    switch (order.kind) {
        case "takeFrom":
            return <>Take from {countryOf(order.country)}</>
        case "takeIn":
            return <>Take in {countryOf(order.country)}</>
        case "takeFromIn":
            return <><CountryFlag code={order.country}/> in {countryOf(order.ground)}</>
        case "bombIn":
            return <><BonusIcon kind="bomb"/> {countryOf(order.country)}</>
        case "breakShields":
            return <><BonusIcon kind="shields"/> {countryOf(order.country)}</>
        case "takeFor":
            return countryOf(order.country)
        case "shieldTiles":
        case "catchBoxes":
        case "answerQuizzes":
        case "fortifyTiles":
            return orderOf(order)
    }
}

function countryOf(code: string): ReactNode {
    return <><CountryFlag code={code}/> {Countries.get(code)?.name ?? code}</>
}
