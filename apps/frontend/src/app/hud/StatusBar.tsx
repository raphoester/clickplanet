import {ReactNode, useState} from "react"
import CountryFlag from "../components/CountryFlag.tsx"
import HomeLogo from "../components/HomeLogo.tsx"
import {ShopIcon} from "../components/icons.tsx"
import {useEdge} from "../components/useEdge.ts"
import {useCountry} from "../countryStore.ts"
import {useShop} from "../shop/shopStore.ts"
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
    const selling = useShop((store) => store.plans.length > 0)
    const setShopOpen = useScreen((screen) => screen.setShopOpen)

    const name = translator().country(country.code)

    // Held sideways the bar is two pills over the globe, this lead and the trailing chip: see docs/screen.md.
    return <header ref={setBar} className="status-bar panel">
        <div className="status-bar-lead">
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
            {selling && <button type="button"
                                className="icon-button status-bar-shop"
                                aria-label={t("menu.shop")}
                                aria-haspopup="dialog"
                                onClick={() => setShopOpen(true)}>
                <ShopIcon/>
            </button>}
        </div>

        {trailing}
    </header>
}
