import {ReactNode} from "react"
import {MissionOrder, TileOrder} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {orderOf} from "../../domain/missions.ts"
import CountryFlag from "../components/CountryFlag.tsx"

export type OrderParts = {
    line: ReactNode
    progress: ReactNode
}

export function partsOf(order: MissionOrder): OrderParts {
    switch (order.kind) {
        case "takeFor":
        case "takeFrom":
        case "takeIn":
            return tilePartsOf(order)
    }
}

function tilePartsOf(order: TileOrder): OrderParts {
    const country = Countries.get(order.country)
    const reached = Math.min(order.taken, order.goal)

    return {
        line: <>{orderOf(order)} <CountryFlag code={order.country}/> {country?.name ?? order.country}</>,
        progress: <>
            <div className="mission-bar"
                 role="progressbar"
                 aria-valuemin={0}
                 aria-valuemax={order.goal}
                 aria-valuenow={reached}
                 aria-label={orderOf(order)}>
                <div className="mission-bar-filled" style={{width: `${reached / order.goal * 100}%`}}/>
            </div>
            <span className="mission-count">{reached}/{order.goal}</span>
        </>,
    }
}
