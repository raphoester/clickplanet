import {ReactNode} from "react"
import {Ask} from "../../domain/nudges.ts"
import {TIKTOK_HANDLE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../../links.ts"
import BuyMeACoffee from "../components/BuyMeACoffee.tsx"
import {TikTokIcon, YouTubeIcon} from "../components/icons.tsx"
import Modal from "../components/Modal.tsx"
import "./AskNudge.css"

export type AskNudgeProps = {
    ask: Ask
    onOpen: () => void
    onClose: () => void
}

export default function AskNudge({ask, onOpen, onClose}: AskNudgeProps) {
    switch (ask) {
        case "coffee":
            return <Modal title="Like ClickPlanet?" footer={<BuyMeACoffee onOpen={onOpen}/>} onClose={onClose}>
                <div className="center-align">
                    <img alt="ClickPlanet logo"
                         src="/static/logo.svg"
                         width="64px"
                         height="auto"/>
                </div>
                <div className="ask-nudge-text">
                    <p>It's free and open source.</p>
                    <p>The servers cost money. Each coffee helps pay for them.</p>
                </div>
            </Modal>
        case "tiktok":
            return <Follow title="ClickPlanet on TikTok"
                           icon={<TikTokIcon size={48}/>}
                           href={TIKTOK_PROFILE}
                           action={`Follow ${TIKTOK_HANDLE}`}
                           onOpen={onOpen}
                           onClose={onClose}/>
        case "youtube":
            return <Follow title="ClickPlanet on YouTube"
                           icon={<YouTubeIcon size={48}/>}
                           href={YOUTUBE_CHANNEL}
                           action="Subscribe"
                           onOpen={onOpen}
                           onClose={onClose}/>
    }
}

type FollowProps = {
    title: string
    icon: ReactNode
    href: string
    action: string
    onOpen: () => void
    onClose: () => void
}

function Follow({title, icon, href, action, onOpen, onClose}: FollowProps) {
    const link = <a href={href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="button button-action ask-nudge-follow"
                    onClick={onOpen}>
        {action}
    </a>
    return <Modal title={title} footer={link} onClose={onClose}>
        <div className="center-align ask-nudge-icon">{icon}</div>
        <div className="ask-nudge-text">
            <p>The biggest wars of the day, in short videos.</p>
            <p>Your flag could be in the next one.</p>
        </div>
    </Modal>
}
