import {ReactNode} from "react"
import {FlagMissionOrder, MissionOrder} from "../../backends/missions.ts"
import {Moved} from "../../domain/missionProgress.ts"
import {reachedOf} from "../../domain/missions.ts"
import {t} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {quizShown, useQuiz} from "../quiz/quizStore.ts"
import {countryOf, orderText} from "./orderParts.tsx"
import "../news/BombNews.css"
import "./Missions.css"

export default function MissionProgress({moved}: {moved: Moved}) {
    const lowered = useQuiz(quizShown)
    const reached = reachedOf(moved.order)

    return <div className={`bomb-news mission-progress${lowered ? " bomb-news--lowered" : ""}`} role="status">
        <div className="bomb-news-line mission-progress-line panel">
            {moved.flag && <span className="mission-progress-kicker">{t("mission.flagMission")}</span>}
            <span className="mission-progress-order">{lineOf(moved.order)}</span>
            <div className="mission-bar mission-progress-bar"
                 role="progressbar"
                 aria-valuemin={0}
                 aria-valuemax={moved.order.goal}
                 aria-valuenow={reached}
                 aria-label={orderText(moved.order)}>
                <div className="mission-bar-filled" style={{width: `${reached / moved.order.goal * 100}%`}}/>
            </div>
            <span className="mission-progress-count">{t("mission.reachedOfGoal", {reached, goal: moved.order.goal})}</span>
        </div>
    </div>
}

function lineOf(order: MissionOrder | FlagMissionOrder): ReactNode {
    switch (order.kind) {
        case "takeFrom":
            return rich("mission.progress.takeFrom", {flag: countryOf(order.country)})
        case "takeIn":
            return rich("mission.progress.takeIn", {place: countryOf(order.country)})
        case "takeFromIn":
            return rich("mission.progress.takeFromIn", {flag: <CountryFlag code={order.country}/>, place: countryOf(order.ground)})
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
            return orderText(order)
    }
}
