import {ON_CRAZYGAMES} from "../../platform.ts"
import "./HomeLogo.css"

// The logo leads back to the home page: to /#home, the one address of it that does not send a returning player
// straight back to the game. CrazyGames shows no home page, so there it is only the logo.
export default function HomeLogo({size, className}: {size: number, className?: string}) {
    const logo = <img alt={ON_CRAZYGAMES ? "ClickPlanet" : ""}
                      src="/static/logo.svg"
                      className={className}
                      width={size}
                      height={size}/>
    if (ON_CRAZYGAMES) return logo

    return <a href="/#home" className="home-logo" aria-label="ClickPlanet home page" title="Home page">{logo}</a>
}
