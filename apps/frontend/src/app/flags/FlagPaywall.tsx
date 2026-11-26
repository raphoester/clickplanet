import {Shop} from "../../backends/shop.ts"
import {changeable, shownOf, Submission} from "../../domain/flagSubmissions.ts"
import {payable} from "../../domain/shop.ts"
import {t} from "../../i18n/page.ts"
import Modal from "../components/Modal.tsx"
import PlanOffer from "../shop/PlanOffer.tsx"
import {useShop} from "../shop/shopStore.ts"
import {capitalOf, coordinatesOf} from "./coordinates.ts"
import FlagCard from "./FlagCard.tsx"
import {Draft, useFlagForm} from "./flagFormStore.ts"
import FlagNote from "./FlagNote.tsx"
import {useFlags} from "./flagsStore.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import {appealOf} from "./appeal.ts"
import SubmissionLine from "./SubmissionLine.tsx"
import {useFlagSubmissions, useVerdictReads} from "./submissionsStore.ts"
import {FlagShooter, useMapShot} from "./mapShot.ts"
import {usePreview} from "./usePreview.ts"
import "./FlagSubmission.css"

export type FlagPaywallProps = {
    // As sent or found: a newer read of it wins.
    submission: Submission
    typed?: Draft
    // The map at the capital with the flag on it.
    shooter: FlagShooter
    shop: Shop
    // Stripe's page.
    toCheckout: (url: string) => void
}

// Step 2: the flag made, its review's verdict as it comes, and the plans once. Pay never waits for the verdict; a
// rejection shows before any payment.
export default function FlagPaywall({submission: sent, typed, shooter, shop, toCheckout}: FlagPaywallProps) {
    const read = useFlagSubmissions((store) => store.submissions.find(({id}) => id === sent.id))
    const back = useFlagForm((form) => form.back)
    const close = useFlagForm((form) => form.close)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const submission = read ?? sent
    const preview = usePreview(typed?.file)
    const shot = useMapShot(shooter, typed?.pinned?.pin ?? submission.pin, typed?.file ?? submission.picture)
    useVerdictReads(submission.state === "waiting")

    const shown = shownOf(submission, catalog)
    const selling = plans.length > 0
    const offered = selling && payable(submission)
    const toChange = changeable(submission) ? () => back(submission) : undefined
    const capital = typed?.pinned ? capitalOf(typed.pinned) : submission.pin && coordinatesOf(submission.pin)

    const footer = submission.state === "rejected"
        ? <button type="button" className="button button-action support-wide" onClick={() => back(submission)}>{t("flagSubmission.fix")}</button>
        : toChange && !offered
            ? <button type="button" className="button button-secondary support-wide" onClick={toChange}>{t("common.back")}</button>
            : undefined

    return <Modal title={t("flagSubmission.yours")} footer={footer} onClose={close} className={offered ? "shop-modal" : undefined}>
        <div className="flag-status">
            <FlagCard name={submission.name}
                      shot={shot}
                      picture={submission.picture?.url ?? preview}
                      capital={capital}
                      chip={<SubmissionStateChip state={shown.state}/>}/>
            <p className="flag-status-line" role="status"><SubmissionLine shown={shown} selling={selling}/></p>
            {submission.state === "rejected" && submission.rejection?.note && <FlagNote text={submission.rejection.note}/>}
            {submission.state === "rejected" && <a className="flag-status-appeal" href={appealOf(submission)}>{t("flagSubmission.appeal")}</a>}
            {offered && <PlanOffer key={submission.id}
                                   shop={shop}
                                   plans={plans}
                                   target={{kind: "submission", id: submission.id}}
                                   toCheckout={toCheckout}
                                   onBack={toChange}/>}
        </div>
    </Modal>
}
