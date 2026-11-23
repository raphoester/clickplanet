import {t} from "../../i18n/page.ts"
import "./HomeLogo.css"

// The logo leads back to the home page: to /#home, the one address of it that does not send a returning player
// straight back to the game.
export default function HomeLogo({size, className}: {size: number, className?: string}) {
    return <a href="/#home" className="home-logo" aria-label={t("menu.logoHome")} title={t("menu.homePage")}>
        <img alt="" src="/static/logo.svg" className={className} width={size} height={size}/>
    </a>
}
