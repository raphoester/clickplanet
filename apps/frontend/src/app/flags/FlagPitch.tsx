import {t} from "../../i18n/page.ts"
import {CheckIcon, FlagIcon} from "../components/icons.tsx"
import "../shop/Shop.css"

const PERKS = ["shop.perk.globe", "shop.perk.players", "shop.perk.chat"] as const

// What a region's flag buys, over the flag made: the shop's door opens here.
export default function FlagPitch() {
    return <div className="flag-pitch">
        <span className="shop-coin"><FlagIcon size={26}/></span>
        <span className="flag-pitch-text">
            <span className="flag-pitch-line">{t("shop.flag.pitch")}</span>
            <ul className="shop-perks">
                {PERKS.map((perk) => <li key={perk}>
                    <CheckIcon size={12}/>
                    <span>{t(perk)}</span>
                </li>)}
            </ul>
        </span>
    </div>
}
