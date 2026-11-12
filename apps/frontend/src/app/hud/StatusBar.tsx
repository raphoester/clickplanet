import {ReactNode, useState} from "react"
import CountryFlag from "../components/CountryFlag.tsx"
import HomeLogo from "../components/HomeLogo.tsx"
import {useEdge} from "../components/useEdge.ts"
import {useCountry} from "../countryStore.ts"
import {useScreen} from "./screenStore.ts"
import {t, translator} from "../../i18n/page.ts"
import "./StatusBar.css"

export const STATUS_BOTTOM = "--status-bottom"

export type StatusBarProps = {
    rank: number | null
    trailing?: ReactNode
}

export default function StatusBar({rank, trailing}: StatusBarProps) {
    const country = useCountry((store) => store.country)
    const [bar, setBar] = useState<HTMLElement | null>(null)
    useEdge(bar, STATUS_BOTTOM, "bottom")
    const boardOpen = useScreen((screen) => screen.sheet === "board")
    const toggleSheet = useScreen((screen) => screen.toggleSheet)

    const name = translator().country(country.code)

    return <header ref={setBar} className="status-bar panel">
        <HomeLogo size={34} className="status-bar-logo"/>

        <button type="button"
                className="status-bar-country"
                aria-label={rank === null ? t("hud.countryNoRank", {country: name}) : t("hud.countryRank", {country: name, rank})}
                aria-expanded={boardOpen}
                onClick={() => toggleSheet("board")}>
            <CountryFlag code={country.code}/>
            <span className="status-bar-name">{name}</span>
            <span className="panel-box status-bar-rank" aria-hidden="true">{rank === null ? "—" : t("board.rankNumber", {rank})}</span>
        </button>

        {trailing}
    </header>
}
