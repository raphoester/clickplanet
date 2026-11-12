import {ReactNode} from "react"
import {Ask} from "../../domain/nudges.ts"
import {TIKTOK_HANDLE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../../links.ts"
import BuyMeACoffee from "../components/BuyMeACoffee.tsx"
import {TikTokIcon, YouTubeIcon} from "../components/icons.tsx"
import Modal from "../components/Modal.tsx"
import {t} from "../../i18n/page.ts"
import "./AskNudge.css"

export type AskNudgeProps = {
    ask: Ask
    onOpen: () => void
    onClose: () => void
}

export default function AskNudge({ask, onOpen, onClose}: AskNudgeProps) {
    switch (ask) {
        case "coffee":
            return <Modal title={t("nudge.coffee.title")} footer={<BuyMeACoffee onOpen={onOpen}/>} onClose={onClose}>
                <div className="center-align">
                    <img alt={t("nudge.coffee.logo")}
                         src="/static/logo.svg"
                         width="64px"
                         height="auto"/>
                </div>
                <div className="ask-nudge-text">
                    <p>{t("nudge.coffee.free")}</p>
                    <p>{t("nudge.coffee.servers")}</p>
                </div>
            </Modal>
        case "tiktok":
            return <Follow title={t("nudge.tiktok.title")}
                           icon={<TikTokIcon size={48}/>}
                           href={TIKTOK_PROFILE}
                           action={t("nudge.tiktok.follow", {handle: TIKTOK_HANDLE})}
                           onOpen={onOpen}
                           onClose={onClose}/>
        case "youtube":
            return <Follow title={t("nudge.youtube.title")}
                           icon={<YouTubeIcon size={48}/>}
                           href={YOUTUBE_CHANNEL}
                           action={t("nudge.youtube.subscribe")}
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
            <p>{t("nudge.follow.videos")}</p>
            <p>{t("nudge.follow.yourFlag")}</p>
        </div>
    </Modal>
}
