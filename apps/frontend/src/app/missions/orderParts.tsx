import {ReactNode} from "react"
import {FlagMissionOrder, MissionOrder} from "../../backends/missions.ts"
import {reachedOf} from "../../domain/missions.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {Key} from "../../i18n/translator.ts"
import CountryFlag from "../components/CountryFlag.tsx"

type Order = MissionOrder | FlagMissionOrder

export type OrderParts = {
    line: ReactNode
    progress: ReactNode
}

const ORDER_LINES: Record<Order["kind"], Key> = {
    takeFor: "mission.order.takeFor",
    takeFrom: "mission.order.takeFrom",
    takeIn: "mission.order.takeIn",
    takeFromIn: "mission.order.takeFromIn",
    bombIn: "mission.order.bombIn",
    shieldTiles: "mission.order.shieldTiles",
    catchBoxes: "mission.order.catchBoxes",
    answerQuizzes: "mission.order.answerQuizzes",
    fortifyTiles: "mission.order.fortifyTiles",
    breakShields: "mission.order.breakShields",
}

export function partsOf(order: Order): OrderParts {
    return {line: rich(ORDER_LINES[order.kind], slotsOf(order, countryOf)), progress: progressOf(order)}
}

export function orderText(order: Order): string {
    return t(ORDER_LINES[order.kind], slotsOf(order, (code) => translator().country(code)))
}

function slotsOf<T>(order: Order, country: (code: string) => T): Record<string, T | number> {
    switch (order.kind) {
        case "takeFor":
        case "takeFrom":
        case "breakShields":
            return {goal: order.goal, flag: country(order.country)}
        case "takeIn":
        case "bombIn":
            return {goal: order.goal, place: country(order.country)}
        case "takeFromIn":
            return {goal: order.goal, flag: country(order.country), place: country(order.ground)}
        case "shieldTiles":
            return {goal: order.goal, shields: order.shields}
        case "catchBoxes":
        case "answerQuizzes":
        case "fortifyTiles":
            return {goal: order.goal}
    }
}

export function countryOf(code: string): ReactNode {
    return <><CountryFlag code={code}/> {translator().country(code)}</>
}

function progressOf(order: Order): ReactNode {
    const reached = reachedOf(order)

    return <>
        <div className="mission-bar"
             role="progressbar"
             aria-valuemin={0}
             aria-valuemax={order.goal}
             aria-valuenow={reached}
             aria-label={orderText(order)}>
            <div className="mission-bar-filled" style={{width: `${reached / order.goal * 100}%`}}/>
        </div>
        <span className="mission-count">{t("mission.reachedOfGoal", {reached, goal: order.goal})}</span>
    </>
}
