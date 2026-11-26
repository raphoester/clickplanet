import {ReactNode, useEffect, useState} from "react"
import {Shop} from "../../backends/shop.ts"
import {changeable, Held, Submission} from "../../domain/flagSubmissions.ts"
import {FlagTime, flagRenewal, flagTimeOf, payable} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {FlagIcon} from "../components/icons.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import {useShop} from "../shop/shopStore.ts"
import {appealOf} from "./appeal.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagNote from "./FlagNote.tsx"
import {useFlags} from "./flagsStore.ts"
import {ChipState, SubmissionStateChip} from "./SubmissionStateChip.tsx"
import SubmissionLine from "./SubmissionLine.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../support/Support.css"
import "./FlagSubmission.css"

export type FlagHubProps = {
    shop: Shop
}

// The shop's door: a new flag, each flag the account holds with its time, and each flag it sent that did not open.
// Each row's one action opens its own page: never the plans in the list.
export default function FlagHub({shop}: FlagHubProps) {
    const submissions = useFlagSubmissions((store) => store.submissions)
    const held = useFlagSubmissions((store) => store.held)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const renewed = useShop((store) => store.renewed)
    const addNew = useFlagForm((form) => form.addNew)
    const payFor = useFlagForm((form) => form.payFor)
    const compose = useFlagForm((form) => form.compose)
    const showPaywall = useFlagForm((form) => form.showPaywall)
    const close = useFlagForm((form) => form.close)
    const [stopping, setStopping] = useState<string>()
    const [stopFailed, setStopFailed] = useState(false)
    const selling = plans.length > 0
    const sent = submissions.filter(({state}) => state !== "opened")

    useEffect(() => {
        useFlagSubmissions.getState().refresh()
        useShop.getState().refresh()
    }, [])

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
        if (!selling) return undefined
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
                return <button type="button"
                               className="button button-mini button-secondary"
                               aria-haspopup="dialog"
                               onClick={() => payFor(flag, true)}>
                    {t("shop.addTime")}
                </button>
            case "removed":
            case "none":
                return undefined
        }
    }

    const heldRow = ({code, removalNote}: Held) => {
        const sold = catalog.soldOf(code)
        const time = flagTimeOf(sold, catalog.closedFlag(code), flagRenewal(renewals, code, submissions))
        const line = time.kind === "removed" || selling ? flagTimeLine(time) : undefined
        const chip: ChipState = sold?.removed ? "removed" : catalog.closedFlag(code) ? "closed" : "opened"
        const action = timeActionOf(code, time)
        return <li key={code} className="panel-box flag-founded">
            <span className="flag-founded-flag"><CountryFlag code={code}/></span>
            <span className="flag-founded-text">
                <span className="flag-founded-name">{translator().country(code)}</span>
                {line && <span className="flag-founded-time">{line}</span>}
            </span>
            <SubmissionStateChip state={chip}/>
            {action && <span className="flag-founded-actions">{action}</span>}
            {sold?.removed && removalNote && <FlagNote text={removalNote}/>}
        </li>
    }

    const sentAction = (submission: Submission): ReactNode => {
        const pay = selling && payable(submission)
        if (!pay && !changeable(submission)) return undefined
        return <span className="flag-founded-actions">
            {changeable(submission) && <button type="button"
                                               className="button button-mini button-secondary"
                                               aria-haspopup="dialog"
                                               onClick={() => compose({
                                                   name: submission.name,
                                                   pinned: submission.pin && {pin: submission.pin},
                                                   replaces: submission.id,
                                               })}>
                {t("flagSubmission.change")}
            </button>}
            {pay && <button type="button"
                            className="button button-mini button-action"
                            aria-haspopup="dialog"
                            onClick={() => showPaywall(submission)}>
                {t("shop.pay")}
            </button>}
        </span>
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
        {sentAction(submission)}
        {submission.state === "rejected" && submission.rejection?.note && <FlagNote text={submission.rejection.note}/>}
        {submission.state === "rejected" && <a className="flag-status-appeal flag-founded-appeal" href={appealOf(submission)}>
            {t("flagSubmission.appeal")}
        </a>}
    </li>

    return <Modal title={t("flagSubmission.founded")} onClose={close}>
        <div className="flag-hub">
            <button type="button" className="button button-action support-wide" aria-haspopup="dialog" onClick={addNew}>
                <FlagIcon size={20}/>
                {t("flagSubmission.addNew")}
            </button>
            <ul className="flag-founded-list">
                {held.map(heldRow)}
                {sent.map(sentRow)}
            </ul>
            {stopFailed && <p className="support-notice" role="alert">{t("shop.stopFailed")}</p>}
        </div>
    </Modal>
}
