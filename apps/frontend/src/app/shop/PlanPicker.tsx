import {useId, useState} from "react"
import {Shop, ShopRefusal, ShopRefusedError} from "../../backends/shop.ts"
import type {CheckoutTarget, Plan, PlanPrice} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"
import "../support/Support.css"
import "./Shop.css"

export type PlanPickerProps = {
    shop: Shop
    // None: no plan and no pay button.
    plans: readonly PlanPrice[]
    target: CheckoutTarget
    // Stripe's page.
    toCheckout: (url: string) => void
    // Back to what was there before the plans.
    onBack?: () => void
}

const NAMES: Record<Plan, Key> = {
    week: "shop.plan.week",
    month: "shop.plan.month",
    year: "shop.plan.year",
}

const FAILURES: Record<ShopRefusal | "failed", Key> = {
    "not-yours": "shop.refusal.notYours",
    "not-found": "shop.refusal.notFound",
    "unavailable": "shop.refusal.unavailable",
    "invalid": "shop.refusal.invalid",
    "failed": "shop.refusal.failed",
}

// In the currency's smallest unit; whole amounts with no cents.
function priceOf(amount: number, currency: string): string {
    const digits = amount % 100 === 0 ? 0 : 2
    return translator().number(amount / 100, {
        style: "currency",
        currency: currency.toUpperCase(),
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
    })
}

function priceLine({plan, amount, currency, renews}: PlanPrice): string {
    const price = priceOf(amount, currency)
    if (!renews) return price
    switch (plan) {
        case "week":
            return t("shop.price.week", {price})
        case "month":
            return t("shop.price.month", {price})
        case "year":
            return t("shop.price.year", {price})
    }
}

// The plans, then the one picked with the buyer's consent that the flag starts now, before Stripe's page.
export default function PlanPicker({shop, plans, target, toCheckout, onBack}: PlanPickerProps) {
    const [picked, setPicked] = useState<PlanPrice>()
    const [agreed, setAgreed] = useState(false)
    const [paying, setPaying] = useState(false)
    const [failure, setFailure] = useState<ShopRefusal | "failed">()
    const consentId = useId()

    if (plans.length === 0) return null

    if (!picked) {
        return <div className="shop-picker">
            <ul className="shop-plans">
                {plans.map((price) => <li key={price.plan}>
                    <button type="button"
                            className="panel-box menu-tile shop-plan"
                            onClick={() => {
                                setPicked(price)
                                setAgreed(false)
                                setFailure(undefined)
                            }}>
                        <span className="shop-plan-name">{t(NAMES[price.plan])}</span>
                        <span className="shop-plan-price">{priceLine(price)}</span>
                    </button>
                </li>)}
            </ul>
            {onBack && <div className="shop-actions">
                <button type="button" className="button button-mini button-secondary" onClick={onBack}>{t("common.back")}</button>
            </div>}
        </div>
    }

    const pay = async () => {
        if (!agreed || paying) return
        setPaying(true)
        setFailure(undefined)
        try {
            toCheckout(await shop.checkout({plan: picked.plan, target, startsNow: true}))
        } catch (e) {
            if (!(e instanceof ShopRefusedError)) console.error("could not open the payment page", e)
            setFailure(e instanceof ShopRefusedError ? e.refusal : "failed")
            setPaying(false)
        }
    }

    return <div className="shop-picker" aria-busy={paying}>
        <div className="panel-box shop-plan shop-plan--picked">
            <span className="shop-plan-name">{t(NAMES[picked.plan])}</span>
            <span className="shop-plan-price">{priceLine(picked)}</span>
        </div>
        <label className="shop-consent" htmlFor={consentId}>
            <input id={consentId}
                   type="checkbox"
                   checked={agreed}
                   disabled={paying}
                   onChange={(e) => setAgreed(e.target.checked)}/>
            <span>{t("shop.consent")}</span>
        </label>
        <p className="shop-legal">
            <a href="/sale" target="_blank" rel="noopener noreferrer">{t("shop.sale")}</a>
            <a href="/refunds" target="_blank" rel="noopener noreferrer">{t("shop.refunds")}</a>
        </p>
        <div className="shop-actions">
            <button type="button"
                    className="button button-secondary"
                    disabled={paying}
                    onClick={() => setPicked(undefined)}>
                {t("common.back")}
            </button>
            <button type="button"
                    className="button button-action shop-pay"
                    disabled={!agreed || paying}
                    onClick={() => void pay()}>
                {paying ? t("common.loading") : t("shop.pay")}
            </button>
        </div>
        {failure && <p className="support-notice" role="alert">{t(FAILURES[failure])}</p>}
    </div>
}
