import {ReactNode, useEffect, useId, useState} from "react"
import {Shop} from "../../backends/shop.ts"
import {changeable, shownOf, Submission} from "../../domain/flagSubmissions.ts"
import {FlagTime, flagRenewal, flagTimeOf, payable} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Fold from "../components/Fold.tsx"
import {FlagIcon} from "../components/icons.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import PlanOffer from "../shop/PlanOffer.tsx"
import {useShop} from "../shop/shopStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagNote from "./FlagNote.tsx"
import {useFlags} from "./flagsStore.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import {appealOf} from "./appeal.ts"
import SubmissionLine from "./SubmissionLine.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../support/Support.css"
import "./FlagSubmission.css"

export type YourFlagsProps = {
    shop: Shop
    // Stripe's page.
    toCheckout: (url: string) => void
    // The flag's way opens: one dialog at a time.
    onAway?: () => void
    // The phone's You sheet folds its sections.
    folded?: boolean
}

// In the profile: each flag the account sent, and each it founded with its time. None changes once paid or decided.
export default function YourFlags({shop, toCheckout, onAway, folded}: YourFlagsProps) {
    const submissions = useFlagSubmissions((store) => store.submissions)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const renewed = useShop((store) => store.renewed)
    const compose = useFlagForm((form) => form.compose)
    const showPaywall = useFlagForm((form) => form.showPaywall)
    const [now] = useState(() => Date.now())
    const [stopping, setStopping] = useState<string>()
    const [stopFailed, setStopFailed] = useState(false)
    const [addingTo, setAddingTo] = useState<string>()
    const titleId = useId()
    const selling = plans.length > 0

    useEffect(() => {
        useFlagSubmissions.getState().refresh()
        useShop.getState().refresh()
    }, [])

    if (submissions.length === 0) return null

    const away = (then: () => void) => {
        onAway?.()
        then()
    }

    const stop = async (id: string) => {
        setStopping(id)
        setStopFailed(false)
        try {
            renewed(await shop.stopRenewal(id))
        } catch (e) {
            console.error("could not stop the renewal", e)
            setStopFailed(true)
        } finally {
            setStopping(undefined)
        }
    }

    const timeActionOf = (flag: string, time: FlagTime): ReactNode => {
        switch (time.kind) {
            case "renews":
                return <button type="button"
                               className="button button-mini button-secondary"
                               disabled={stopping !== undefined}
                               onClick={() => void stop(time.renewal.id)}>
                    {t("shop.stopRenewal")}
                </button>
            // A renewal stopping charges no more: more time is no second charge.
            case "stopping":
            case "open":
            case "closed":
                return <button type="button" className="button button-mini button-secondary" onClick={() => setAddingTo(flag)}>
                    {t("shop.addTime")}
                </button>
            case "none":
                return undefined
        }
    }

    const foundedRow = (submission: Submission) => {
        const {flag} = submission
        const closed = catalog.closedFlag(flag)
        const time: FlagTime = selling
            ? flagTimeOf(catalog.soldOf(flag), closed, flagRenewal(renewals, flag, submissions), now)
            : {kind: "none"}
        const line = flagTimeLine(time)
        const action = timeActionOf(flag, time)
        return <li key={submission.id} className="panel-box flag-founded">
            <span className="flag-founded-flag"><CountryFlag code={flag}/></span>
            <span className="flag-founded-text">
                <span className="flag-founded-name">{translator().country(flag)}</span>
                {line && <span className="flag-founded-time">{line}</span>}
            </span>
            <SubmissionStateChip state={shownOf(submission, catalog).state}/>
            {action && <span className="flag-founded-actions">{action}</span>}
            {closed && submission.closeNote && <FlagNote text={submission.closeNote}/>}
        </li>
    }

    const sentRow = (submission: Submission) => <li key={submission.id} className="panel-box flag-founded">
        <span className="flag-founded-flag">
            {submission.picture ? <img src={submission.picture.url} alt=""/> : <FlagIcon size={22}/>}
        </span>
        <span className="flag-founded-text">
            <span className="flag-founded-name">{submission.name}</span>
            <span className="flag-founded-time"><SubmissionLine shown={submission} selling={selling}/></span>
        </span>
        <SubmissionStateChip state={submission.state}/>
        {(changeable(submission) || (selling && payable(submission))) && <span className="flag-founded-actions">
            {changeable(submission) && <button type="button"
                                               className="button button-mini button-secondary"
                                               aria-haspopup="dialog"
                                               onClick={() => away(() => compose({
                                                   name: submission.name,
                                                   pinned: submission.pin && {pin: submission.pin},
                                                   replaces: submission.id,
                                               }))}>
                {t("flagSubmission.change")}
            </button>}
            {selling && payable(submission) && <button type="button"
                                                       className="button button-mini button-action"
                                                       aria-haspopup="dialog"
                                                       onClick={() => away(() => showPaywall(submission))}>
                {t("shop.pay")}
            </button>}
        </span>}
        {submission.state === "rejected" && submission.rejection?.note && <FlagNote text={submission.rejection.note}/>}
        {submission.state === "rejected" && <a className="flag-status-appeal flag-founded-appeal" href={appealOf(submission)}>
            {t("flagSubmission.appeal")}
        </a>}
    </li>

    const body = addingTo !== undefined && selling
        ? <div className="flag-status">
            <div className="panel-box flag-founded">
                <span className="flag-founded-flag"><CountryFlag code={addingTo}/></span>
                <span className="flag-founded-name">{translator().country(addingTo)}</span>
            </div>
            <PlanOffer shop={shop}
                       plans={plans}
                       target={{kind: "flag", flag: addingTo}}
                       toCheckout={toCheckout}
                       onBack={() => setAddingTo(undefined)}/>
        </div>
        : <>
            <ul className="flag-founded-list" aria-labelledby={folded ? undefined : titleId}>
                {submissions.map((submission) => submission.state === "opened" ? foundedRow(submission) : sentRow(submission))}
            </ul>
            {stopFailed && <p className="support-notice" role="alert">{t("shop.stopFailed")}</p>}
        </>

    if (folded) return <Fold title={t("flagSubmission.founded")} startOpen>{body}</Fold>
    return <section className="profile-section your-flags">
        <h3 className="profile-section-title" id={titleId}>{t("flagSubmission.founded")}</h3>
        {body}
    </section>
}
