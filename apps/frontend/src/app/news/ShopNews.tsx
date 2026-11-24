import {useEffect, useRef} from "react"
import type {ShopReturn} from "../../domain/shopReturn.ts"
import {t} from "../../i18n/page.ts"
import "./BombNews.css"
import "./WelcomeNews.css"

export const SHOP_NEWS_MS = 4000

export type ShopNewsProps = {
    back: ShopReturn
    onDone: () => void
}

export default function ShopNews({back, onDone}: ShopNewsProps) {
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), SHOP_NEWS_MS)
        return () => clearTimeout(timer)
    }, [back])

    return <div className="bomb-news" role="status" aria-live="polite">
        <div className="bomb-news-line welcome-news-line panel">
            {back === "paid" ? t("news.shop.paid") : t("news.shop.cancelled")}
        </div>
    </div>
}
