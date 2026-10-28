import {ReactNode} from "react"
import {FlagMissionOrder, MissionOrder, ShieldOrder, TileOrder} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {orderOf, shieldOrderOf} from "../../domain/missions.ts"
import CountryFlag from "../components/CountryFlag.tsx"

export type OrderParts = {
    line: ReactNode
    progress: ReactNode
}

export function partsOf(order: MissionOrder | FlagMissionOrder): OrderParts {
    switch (order.kind) {
        case "takeFor":
        case "takeFrom":
        case "takeIn":
            return tilePartsOf(order)
        case "breakShields":
            return shieldPartsOf(order)
    }
}

function tilePartsOf(order: TileOrder): OrderParts {
    return countedParts(orderOf(order), order.country, order.taken, order.goal)
}

function shieldPartsOf(order: ShieldOrder): OrderParts {
    return countedParts(shieldOrderOf(order), order.country, order.broken, order.goal)
}

function countedParts(text: string, code: string, count: number, goal: number): OrderParts {
    const country = Countries.get(code)
    const reached = Math.min(count, goal)

    return {
        line: <>{text} <CountryFlag code={code}/> {country?.name ?? code}</>,
        progress: <>
            <div className="mission-bar"
                 role="progressbar"
                 aria-valuemin={0}
                 aria-valuemax={goal}
                 aria-valuenow={reached}
                 aria-label={text}>
                <div className="mission-bar-filled" style={{width: `${reached / goal * 100}%`}}/>
            </div>
            <span className="mission-count">{reached}/{goal}</span>
        </>,
    }
}
