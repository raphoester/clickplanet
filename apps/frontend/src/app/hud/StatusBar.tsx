import {ReactNode, useState} from "react"
import {Country} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import HomeLogo from "../components/HomeLogo.tsx"
import {useBottomEdge} from "../components/useBottomEdge.ts"
import {useScreen} from "./screenStore.ts"
import "./StatusBar.css"

export const STATUS_BOTTOM = "--status-bottom"

export type StatusBarProps = {
    country: Country
    rank: number | null
    trailing?: ReactNode
}

export default function StatusBar({country, rank, trailing}: StatusBarProps) {
    const [bar, setBar] = useState<HTMLElement | null>(null)
    useBottomEdge(bar, STATUS_BOTTOM)
    const boardOpen = useScreen((screen) => screen.sheet === "board")
    const toggleSheet = useScreen((screen) => screen.toggleSheet)

    const ranked = rank === null ? "no rank yet" : `rank ${rank}`

    return <header ref={setBar} className="status-bar panel">
        <HomeLogo size={34} className="status-bar-logo"/>

        <button type="button"
                className="status-bar-country"
                aria-label={`${country.name}, ${ranked}. Leaderboard`}
                aria-expanded={boardOpen}
                onClick={() => toggleSheet("board")}>
            <CountryFlag code={country.code}/>
            <span className="status-bar-name">{country.name}</span>
            <span className="panel-box status-bar-rank" aria-hidden="true">{rank === null ? "—" : `#${rank}`}</span>
        </button>

        {trailing}
    </header>
}
