import {INSTAGRAM_PROFILE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../links.ts"
import {DiscordIcon, InstagramIcon, TikTokIcon, YouTubeIcon} from "./components/icons.tsx"
import DiscordButton from "./alerts/DiscordButton.tsx"
import {DiscordOffer} from "./alerts/useDiscordAlerts.ts"
import {ON_CRAZYGAMES} from "../platform.ts"
import {t} from "../i18n/page.ts"
import "./About.css"

const FOLLOW = [
    {name: "TikTok", href: TIKTOK_PROFILE, icon: <TikTokIcon size={20}/>},
    {name: "YouTube", href: YOUTUBE_CHANNEL, icon: <YouTubeIcon size={20}/>},
    {name: "Instagram", href: INSTAGRAM_PROFILE, icon: <InstagramIcon size={20}/>},
]

export type AboutProps = {
    discord?: DiscordOffer
}

export default function About({discord}: AboutProps) {
    return <>
        <h3>{t("about.title")}</h3>
        <h4>{t("about.subtitle")}</h4>
        <p>{t("about.pitch")}</p>
        <div className="about-social">
            <DiscordButton offer={discord} className="button button-mini about-follow">
                <DiscordIcon size={20}/>
            </DiscordButton>
            {FOLLOW.map((place) => <a key={place.name}
                                      className="button button-mini about-follow"
                                      target="_blank"
                                      rel="noopener noreferrer"
                                      href={place.href}
                                      aria-label={place.name}>
                {place.icon}
            </a>)}
        </div>
        <div className="about-author">
            <p className="menu-label">{t("about.createdBy")}</p>
            <div className="center-align">
                <img alt="Raphaël Oester"
                     src="/static/raphael.jpeg"
                     className="about-photo"/>
            </div>
            <h3>Raphaël Oester</h3>
            {!ON_CRAZYGAMES && <p className="center-align">{t("about.freelance")}</p>}
            {!ON_CRAZYGAMES && <div className="about-social">
                <a className="button button-mini" target="_blank" rel="noopener noreferrer" href="https://www.linkedin.com/in/raphael-oester/"><b>in</b></a>
                <a className="button button-mini" target="_blank" rel="noopener noreferrer" href="https://x.com/raphael_oester"><b>X</b></a>
                <a className="button button-mini" target="_blank" rel="noopener noreferrer" href="https://github.com/raphoester"><b>GitHub</b></a>
            </div>}
        </div>
        <p className="about-legal">
            {!ON_CRAZYGAMES && <a href="/#home">{t("menu.homePage")}</a>}
            <a href="/privacy" target="_blank" rel="noopener noreferrer">{t("about.privacy")}</a>
            <a href="/terms" target="_blank" rel="noopener noreferrer">{t("about.terms")}</a>
        </p>
    </>
}
