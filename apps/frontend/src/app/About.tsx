import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../links.ts"
import {DiscordIcon, InstagramIcon, TikTokIcon, YouTubeIcon} from "./components/icons.tsx"
import "./About.css"

const FOLLOW = [
    {name: "Discord", href: DISCORD_INVITE, icon: <DiscordIcon size={20}/>},
    {name: "TikTok", href: TIKTOK_PROFILE, icon: <TikTokIcon size={20}/>},
    {name: "YouTube", href: YOUTUBE_CHANNEL, icon: <YouTubeIcon size={20}/>},
    {name: "Instagram", href: INSTAGRAM_PROFILE, icon: <InstagramIcon size={20}/>},
]

export default function About() {
    return <>
        <h3>The ultimate world war</h3>
        <h4>It’s like Pixel Wars but way more epic</h4>
        <p>ClickPlanet is a virtual battleground where you conquer territories for a country, click after
            click. Out-click rival nations, and dominate the map. Every territory is yours, until someone
            takes it back!</p>
        <div className="about-social">
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
            <p className="menu-label">Created by</p>
            <div className="center-align">
                <img alt="Raphaël Oester"
                     src="/static/raphael.jpeg"
                     className="about-photo"/>
            </div>
            <h3>Raphaël Oester</h3>
            <p className="center-align">Freelance developer, open to new opportunities</p>
            <div className="about-social">
                <a className="button button-mini" target="_blank" rel="noopener noreferrer" href="https://www.linkedin.com/in/raphael-oester/"><b>in</b></a>
                <a className="button button-mini" target="_blank" rel="noopener noreferrer" href="https://x.com/raphael_oester"><b>X</b></a>
                <a className="button button-mini" target="_blank" rel="noopener noreferrer" href="https://github.com/raphoester"><b>GitHub</b></a>
            </div>
        </div>
        <p className="about-legal">
            <a href="/#home">Home page</a>
            <a href="/privacy" target="_blank" rel="noopener noreferrer">Privacy policy</a>
            <a href="/terms" target="_blank" rel="noopener noreferrer">Terms of service</a>
        </p>
    </>
}
