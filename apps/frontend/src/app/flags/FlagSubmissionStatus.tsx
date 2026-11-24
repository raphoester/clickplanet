import {ReactNode, useState} from "react"
import {Shop} from "../../backends/shop.ts"
import type {FlagCatalog} from "../../domain/flags.ts"
import {RejectionReason, Shown, shownOf, Submission, waitingOf} from "../../domain/flagSubmissions.ts"
import {payable} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {Key} from "../../i18n/translator.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {FlagIcon} from "../components/icons.tsx"
import PlanOffer from "../shop/PlanOffer.tsx"
import {useShop} from "../shop/shopStore.ts"
import FlagNote from "./FlagNote.tsx"
import {useFlags} from "./flagsStore.ts"
import {Draft, useFlagForm} from "./flagFormStore.ts"
import FoundedFlags from "./FoundedFlags.tsx"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import "./FlagSubmission.css"

export type FlagSubmissionStatusProps = {
    // The newest first: the first is the one shown.
    submissions: readonly Submission[]
    shop: Shop
    // Stripe's page.
    toCheckout: (url: string) => void
}

const REASONS: Record<RejectionReason, Key> = {
    "not-a-place": "flagSubmission.rejected.notAPlace",
    "exists": "flagSubmission.rejected.exists",
    "unreadable": "flagSubmission.rejected.unreadable",
    "hateful": "flagSubmission.rejected.hateful",
    "other": "flagSubmission.rejected.other",
}

function dateOf(ms: number): string {
    return translator().dateTime(ms, {dateStyle: "medium"})
}

function lineOf(shown: Shown, catalog: FlagCatalog, selling: boolean): ReactNode {
    const flag = <strong>{shown.name}</strong>
    const sold = catalog.soldOf(shown.flag)
    switch (shown.state) {
        case "waiting":
            if (shown.paid) return t("flagSubmission.status.paid")
            return selling ? t("flagSubmission.status.payFirst") : t("flagSubmission.status.waiting", {date: dateOf(shown.submittedAt)})
        case "approved":
            if (shown.paid) return t("flagSubmission.status.opening")
            return selling ? t("flagSubmission.status.pay") : t("flagSubmission.status.approved")
        case "rejected":
            return t(REASONS[shown.rejection?.reason ?? "other"])
        case "opened":
            return sold === undefined
                ? rich("flagSubmission.status.opened", {flag})
                : rich("flagSubmission.status.openUntil", {flag, date: dateOf(sold.endsAt)})
        case "closed":
            return sold?.closedAt === undefined ? undefined : rich("flagSubmission.status.closed", {flag, date: dateOf(sold.closedAt)})
    }
}

// The same submission again: the picture is picked again, since the page keeps none.
function againOf({name, pin}: Submission): Draft {
    return {name, pinned: pin && {pin}}
}

export default function FlagSubmissionStatus({submissions, shop, toCheckout}: FlagSubmissionStatusProps) {
    const compose = useFlagForm((form) => form.compose)
    const close = useFlagForm((form) => form.close)
    const payFailure = useFlagForm((form) => form.payFailure)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const [addingTo, setAddingTo] = useState<string>()
    const [latest] = submissions
    const shown = shownOf(latest, catalog)
    const waiting = waitingOf(submissions)
    const selling = plans.length > 0
    const offered = selling && payable(latest)
    // Pay is the one thing to press while the plans show.
    const footerButton = offered || (selling && addingTo !== undefined) ? "button-secondary" : "button-action"

    // A paid flag is not changed, and no other is sent while it waits.
    const footer = waiting?.paid
        ? undefined
        : waiting
        ? <button type="button" className="button support-wide" onClick={() => compose({...againOf(waiting), replaces: waiting.id})}>
            {t("flagSubmission.change")}
        </button>
        : latest.state === "rejected"
            ? <button type="button" className={`button ${footerButton} support-wide`} onClick={() => compose(againOf(latest))}>{t("common.retry")}</button>
            : <button type="button" className={`button ${footerButton} support-wide`} onClick={() => compose()}>{t("flagSubmission.new")}</button>

    if (addingTo !== undefined && selling) {
        return <Modal title={t("flagSubmission.yours")} footer={footer} onClose={close} className="shop-modal">
            <div className="flag-status">
                <div className="panel-box flag-status-card">
                    <span className="flag-status-picture"><CountryFlag code={addingTo}/></span>
                    <span className="flag-status-name">{translator().country(addingTo)}</span>
                </div>
                <PlanOffer shop={shop}
                            plans={plans}
                            target={{kind: "flag", flag: addingTo}}
                            toCheckout={toCheckout}
                            onBack={() => setAddingTo(undefined)}/>
            </div>
        </Modal>
    }

    return <Modal title={t("flagSubmission.yours")} footer={footer} onClose={close} className={offered ? "shop-modal" : undefined}>
        <div className="flag-status">
            <div className="panel-box flag-status-card">
                <span className="flag-status-picture">
                    {latest.picture
                        ? <img src={latest.picture.url} alt=""/>
                        : latest.state === "opened" ? <CountryFlag code={latest.flag}/> : <FlagIcon size={22}/>}
                </span>
                <span className="flag-status-name">{latest.name}</span>
                <SubmissionStateChip state={shown.state}/>
            </div>
            <p className="flag-status-line" role="status">{lineOf(shown, catalog, selling)}</p>
            {latest.state === "rejected" && latest.rejection?.note && <FlagNote text={latest.rejection.note}/>}
            {shown.state === "closed" && latest.closeNote && <FlagNote text={latest.closeNote}/>}
            {offered && <PlanOffer shop={shop}
                                   plans={plans}
                                   target={{kind: "submission", id: latest.id}}
                                   toCheckout={toCheckout}
                                   failed={payFailure}/>}

            <FoundedFlags submissions={submissions}
                          shop={shop}
                          noteShownFor={shown.state === "closed" ? shown.flag : undefined}
                          onAddTime={setAddingTo}/>
        </div>
    </Modal>
}
