import {ReactNode, useState} from "react"
import {Country} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {useBottomEdge} from "../components/useBottomEdge.ts"
import "./StatusBar.css"

export const STATUS_BOTTOM = "--status-bottom"

export type StatusBarProps = {
    country: Country
    rank: number | null
    boardOpen: boolean
    onOpenBoard: () => void
    trailing?: ReactNode
}

export default function StatusBar({country, rank, boardOpen, onOpenBoard, trailing}: StatusBarProps) {
    const [bar, setBar] = useState<HTMLElement | null>(null)
    useBottomEdge(bar, STATUS_BOTTOM)

    const ranked = rank === null ? "no rank yet" : `rank ${rank}`

    return <header ref={setBar} className="status-bar panel">
        <img alt="ClickPlanet"
             src="/static/logo.svg"
             className="status-bar-logo"
             width="34"
             height="34"/>

        <button type="button"
                className="status-bar-country"
                aria-label={`${country.name}, ${ranked}. Leaderboard`}
                aria-expanded={boardOpen}
                onClick={onOpenBoard}>
            <CountryFlag code={country.code}/>
            <span className="status-bar-name">{country.name}</span>
            <span className="panel-box status-bar-rank" aria-hidden="true">{rank === null ? "—" : `#${rank}`}</span>
        </button>

        {trailing}
    </header>
}
