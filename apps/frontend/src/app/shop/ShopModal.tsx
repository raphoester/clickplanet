import {ReactNode, useEffect} from "react"
import {Shop} from "../../backends/shop.ts"
import {offerOf, pickedOf, ShopOffer} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {CheckIcon, FlagIcon} from "../components/icons.tsx"
import Modal from "../components/Modal.tsx"
import {useFlagForm} from "../flags/flagFormStore.ts"
import {useFlags} from "../flags/flagsStore.ts"
import {SubmissionStateChip} from "../flags/SubmissionStateChip.tsx"
import {useFlagSubmissions} from "../flags/submissionsStore.ts"
import {useScreen} from "../hud/screenStore.ts"
import {flagTimeLine} from "./flagTime.ts"
import PlanCards from "./PlanCards.tsx"
import PlanOffer from "./PlanOffer.tsx"
import {useShop} from "./shopStore.ts"
import "./Shop.css"

export type ShopModalProps = {
    shop: Shop
    // Stripe's page.
    toCheckout: (url: string) => void
}

type Head = {picture: ReactNode, name: string, line: ReactNode}

const STEPS = ["shop.step.send", "shop.step.review", "shop.step.pay"] as const

// Where a new flag is on its way to the globe: sent, checked, paid.
function Steps({at}: {at: number}) {
    return <ol className="shop-steps">
        {STEPS.map((step, i) => <li key={step}
                                    className={i < at ? "shop-step shop-step--done" : i === at ? "shop-step shop-step--now" : "shop-step"}
                                    aria-current={i === at ? "step" : undefined}>
            <span className="shop-step-coin">{i < at ? <CheckIcon size={12}/> : i + 1}</span>
            <span className="shop-step-name">{t(step)}</span>
        </li>)}
    </ol>
}

const STEP_AT: Partial<Record<ShopOffer["kind"], number>> = {new: 0, waiting: 1, open: 2}

const GOLD_FLAG = <span className="shop-coin"><FlagIcon size={30}/></span>

function headOf(offer: ShopOffer): Head {
    switch (offer.kind) {
        case "new":
            return {picture: GOLD_FLAG, name: t("shop.flag.name"), line: t("shop.flag.pitch")}
        case "open":
        case "waiting":
            return {
                picture: offer.submission.picture ? <img src={offer.submission.picture.url} alt=""/> : GOLD_FLAG,
                name: offer.submission.name,
                line: <SubmissionStateChip state={offer.submission.state}/>,
            }
        case "addTime":
        case "renewing":
            return {
                picture: <CountryFlag code={offer.flag}/>,
                name: translator().country(offer.flag),
                line: flagTimeLine(offer.time),
            }
    }
}

// What the game sells. Today the region's flag: the plans side by side, and the one step the account is at, from adding
// its flag to paying for it.
export default function ShopModal({shop, toCheckout}: ShopModalProps) {
    const open = useScreen((screen) => screen.shopOpen)
    const setShopOpen = useScreen((screen) => screen.setShopOpen)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const chosen = useShop((store) => store.chosen)
    const choose = useShop((store) => store.choose)
    const submissions = useFlagSubmissions((store) => store.submissions)
    const catalog = useFlags((store) => store.catalog)
    const showFlagPlace = useFlagForm((form) => form.show)

    useEffect(() => {
        if (!open) return
        useFlagSubmissions.getState().refresh()
        useShop.getState().refresh()
    }, [open])

    if (!open) return null

    const close = () => setShopOpen(false)
    const toFlagPlace = () => {
        close()
        showFlagPlace()
    }
    const offer = offerOf(submissions, catalog, renewals, Date.now())
    const head = headOf(offer)
    const picked = pickedOf(plans, chosen)

    const body = (): ReactNode => {
        if (picked === undefined) return undefined
        switch (offer.kind) {
            case "new":
                return <>
                    <PlanCards plans={plans} picked={picked.plan} onPick={choose}/>
                    <button type="button" className="button button-action shop-go" aria-haspopup="dialog" onClick={toFlagPlace}>
                        {t("shop.flag.add")}
                    </button>
                </>
            case "waiting":
                return <>
                    <PlanCards plans={plans} picked={picked.plan} onPick={choose}/>
                    <button type="button" className="button button-secondary shop-go" aria-haspopup="dialog" onClick={toFlagPlace}>
                        {t("shop.flag.yours")}
                    </button>
                </>
            case "open":
                return <PlanOffer key={offer.submission.id}
                                  shop={shop}
                                  plans={plans}
                                  target={{kind: "submission", id: offer.submission.id}}
                                  toCheckout={toCheckout}/>
            case "addTime":
                return <PlanOffer key={offer.flag}
                                  shop={shop}
                                  plans={plans}
                                  target={{kind: "flag", flag: offer.flag}}
                                  toCheckout={toCheckout}/>
            case "renewing":
                return <button type="button" className="button button-secondary shop-go" aria-haspopup="dialog" onClick={toFlagPlace}>
                    {t("shop.flag.yours")}
                </button>
        }
    }

    return <Modal title={t("menu.shop")} onClose={close} className="shop-modal">
        <section className="shop-product" aria-label={t("shop.flag.name")}>
            <header className="shop-product-head">
                <span className="shop-product-picture">{head.picture}</span>
                <span className="shop-product-title">
                    <h3 className="shop-product-name">{head.name}</h3>
                    {head.line && <span className="shop-product-line">{head.line}</span>}
                </span>
            </header>
            {offer.kind === "new" && <ul className="shop-perks">
                {(["shop.perk.globe", "shop.perk.players", "shop.perk.chat"] as const).map((perk) => <li key={perk}>
                    <CheckIcon size={14}/>
                    <span>{t(perk)}</span>
                </li>)}
            </ul>}
            {picked && STEP_AT[offer.kind] !== undefined && <Steps at={STEP_AT[offer.kind] ?? 0}/>}
            {body()}
        </section>
    </Modal>
}
