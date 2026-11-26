import {CSSProperties, useEffect, useId, useRef} from "react"
import {createPortal} from "react-dom"
import {Sale, salePriceOf, saleRuns} from "../../domain/shop.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {t} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {CloseIcon} from "../components/icons.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {useNow} from "../season/useNow.ts"
import Money from "../shop/Money.tsx"
import {planName, saleLeftOf, savingLine} from "../shop/prices.ts"
import {useShop} from "../shop/shopStore.ts"
import "../shop/Shop.css"
import "../titles/TitleUnlocked.css"
import "./SaleNudge.css"

export type SaleNudgeProps = {
    sale: Sale
    // To the region's flag's flow.
    onOpen: () => void
    onClose: () => void
}

// The shop's sale, full screen: its cut on a gold tag, the three plans at their sale price, and the time it has left.
export default function SaleNudge({sale, onOpen, onClose}: SaleNudgeProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const plans = useShop((store) => store.plans)
    const now = useNow()
    const runs = saleRuns(sale, now)

    useModalDialog(dialog, onClose)

    useEffect(() => {
        if (!runs) onClose()
    }, [runs, onClose])

    const timing = {"--impact": `${TITLE_REVEAL.impact * 1000}ms`} as CSSProperties

    return createPortal(
        <div ref={dialog}
             className="title-reveal sale-nudge"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={timing}>
            <button type="button" className="icon-button sale-nudge-close" aria-label={t("common.close")} onClick={onClose}>
                <CloseIcon/>
            </button>
            <div className="title-reveal-stage sale-nudge-stage">
                <h2 id={headId} className="title-reveal-head sale-nudge-head">{t("nudge.sale.head")}</h2>
                <div className="sale-nudge-burst" aria-hidden="true">
                    <div className="title-reveal-rays sale-nudge-rays"/>
                    <div className="sale-nudge-tag">
                        <span className="sale-nudge-cut">{savingLine(sale.percentOff)}</span>
                    </div>
                </div>
                <p className="sale-nudge-line">{t("nudge.sale.line")}</p>
                <ul className="sale-nudge-plans">
                    {plans.map((price) => <li key={price.plan} className="sale-nudge-plan">
                        <span className="sale-nudge-plan-name">{planName(price.plan)}</span>
                        <Money price={price} className="sale-nudge-was" struck/>
                        <Money price={salePriceOf(price, sale)} className="sale-nudge-now"/>
                    </li>)}
                </ul>
                <p className="sale-nudge-clock" role="timer">
                    {rich("nudge.sale.endsIn", {time: <span className="shop-sale-left">{saleLeftOf(sale, now)}</span>})}
                </p>
                <div className="title-reveal-actions sale-nudge-actions">
                    <button type="button" className="button button-action" onClick={onOpen}>{t("shop.flag.pitch")}</button>
                </div>
                <p className="sale-nudge-fine">{t("nudge.sale.firstPayment")}</p>
            </div>
        </div>,
        document.body,
    )
}
