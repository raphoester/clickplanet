import {Shop} from "../../backends/shop.ts"
import {flagRenewal, flagTimeOf} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import Modal from "../components/Modal.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import PlanOffer from "../shop/PlanOffer.tsx"
import {useShop} from "../shop/shopStore.ts"
import {coordinatesOf} from "./coordinates.ts"
import FlagCard from "./FlagCard.tsx"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import {FlagShooter, useMapShot} from "./mapShot.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "./FlagSubmission.css"

export type FlagPayScreenProps = {
    flag: string
    // Back goes to the account's flags.
    fromHub: boolean
    // The map at the capital with the flag on it.
    shooter: FlagShooter
    shop: Shop
    // Stripe's page.
    toCheckout: (url: string) => void
}

// The pay page of a flag, as a new flag's: its holder adds time, or a player claims one whose time ended. The flag on
// the game's map at its capital, then the plans.
export default function FlagPayScreen({flag, fromHub, shooter, shop, toCheckout}: FlagPayScreenProps) {
    const catalog = useFlags((store) => store.catalog)
    const holds = useFlagSubmissions((store) => store.held.some(({code}) => code === flag))
    const submissions = useFlagSubmissions((store) => store.submissions)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const toHub = useFlagForm((form) => form.toHub)
    const close = useFlagForm((form) => form.close)
    const sold = catalog.soldOf(flag)
    const closed = catalog.closedFlag(flag)
    const shot = useMapShot(shooter, sold?.pin, sold?.picture)
    const name = translator().country(flag)
    const claim = !holds
    const line = holds ? flagTimeLine(flagTimeOf(sold, closed, flagRenewal(renewals, flag, submissions))) : undefined
    const offered = plans.length > 0 && sold !== undefined && !sold.removed && (holds || closed)
    const back = fromHub ? toHub : undefined

    return <Modal title={claim ? t("flagSubmission.claimTitle", {flag: name}) : t("shop.addTime")}
                  onClose={close}
                  className={offered ? "shop-modal" : undefined}
                  footer={!offered && back
                      ? <button type="button" className="button button-secondary support-wide" onClick={back}>{t("common.back")}</button>
                      : undefined}>
        <div className="flag-status">
            <FlagCard name={name}
                      shot={shot}
                      picture={sold?.picture.url}
                      capital={sold?.pin && coordinatesOf(sold.pin)}
                      chip={<SubmissionStateChip state={sold?.removed ? "removed" : closed ? "closed" : "opened"}/>}/>
            {line && <p className="flag-status-line" role="status">{line}</p>}
            {offered && <PlanOffer key={flag}
                                   shop={shop}
                                   plans={plans}
                                   target={{kind: "flag", flag}}
                                   toCheckout={toCheckout}
                                   onBack={back}/>}
        </div>
    </Modal>
}
