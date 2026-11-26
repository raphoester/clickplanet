import {useState} from "react"
import {PayFailure, Shop, ShopRefusedError} from "../../backends/shop.ts"
import {CheckoutTarget, pickedOf, PlanPrice} from "../../domain/shop.ts"
import {t} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"
import {rich} from "../../i18n/rich.tsx"
import "../support/Support.css"
import Money from "./Money.tsx"
import {Consent, FinePrint} from "./PayTerms.tsx"
import PlanCards from "./PlanCards.tsx"
import {useShop} from "./shopStore.ts"
import "./Shop.css"

export type PlanOfferProps = {
    shop: Shop
    // None: no plan and no pay button.
    plans: readonly PlanPrice[]
    target: CheckoutTarget
    // Stripe's page.
    toCheckout: (url: string) => void
    // Back to what was there before the plans.
    onBack?: () => void
}

const FAILURES: Record<PayFailure, Key> = {
    "not-yours": "shop.refusal.notYours",
    "not-found": "shop.refusal.notFound",
    "unavailable": "shop.refusal.unavailable",
    "invalid": "shop.refusal.invalid",
    "failed": "shop.refusal.failed",
}

// The plans, the buyer's consent that the time starts once the flag is approved or at once, and Pay, to Stripe's page.
export default function PlanOffer({shop, plans, target, toCheckout, onBack}: PlanOfferProps) {
    const chosen = useShop((store) => store.chosen)
    const choose = useShop((store) => store.choose)
    const [agreed, setAgreed] = useState(false)
    const [paying, setPaying] = useState(false)
    const [failure, setFailure] = useState<PayFailure>()
    const picked = pickedOf(plans, chosen)

    if (picked === undefined) return null

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

    return <div className="shop-offer" aria-busy={paying}>
        <PlanCards plans={plans}
                   picked={picked.plan}
                   disabled={paying}
                   onPick={(plan) => {
                       choose(plan)
                       setFailure(undefined)
                   }}/>
        <Consent agreed={agreed} disabled={paying} onAgree={setAgreed}/>
        <div className="shop-actions">
            {onBack && <button type="button" className="button button-secondary" disabled={paying} onClick={onBack}>
                {t("common.back")}
            </button>}
            <button type="button"
                    className="button button-action shop-pay"
                    disabled={!agreed || paying}
                    onClick={() => void pay()}>
                {paying ? t("common.loading") : rich("shop.payPrice", {price: <Money price={picked}/>})}
            </button>
        </div>
        {failure && <p className="support-notice" role="alert">{t(FAILURES[failure])}</p>}
        <FinePrint renews={picked.renews}/>
    </div>
}
