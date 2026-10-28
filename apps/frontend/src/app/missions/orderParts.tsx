import {ReactNode} from "react"
import {MissionOrder} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {orderOf, reachedOf} from "../../domain/missions.ts"
import CountryFlag from "../components/CountryFlag.tsx"

export type OrderParts = {
    line: ReactNode
    progress: ReactNode
}

export function partsOf(order: MissionOrder): OrderParts {
    return {line: lineOf(order), progress: progressOf(order)}
}

function lineOf(order: MissionOrder): ReactNode {
    switch (order.kind) {
        case "takeFor":
        case "takeFrom":
        case "takeIn":
        case "bombIn":
            return <>{orderOf(order)} {countryOf(order.country)}</>
        case "takeFromIn":
            return <>{orderOf(order)} {countryOf(order.country)} in {countryOf(order.ground)}</>
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

function progressOf(order: MissionOrder): ReactNode {
    const reached = reachedOf(order)

    return <>
        <div className="mission-bar"
             role="progressbar"
             aria-valuemin={0}
             aria-valuemax={order.goal}
             aria-valuenow={reached}
             aria-label={orderOf(order)}>
            <div className="mission-bar-filled" style={{width: `${reached / order.goal * 100}%`}}/>
        </div>
        <span className="mission-count">{reached}/{order.goal}</span>
    </>
}
