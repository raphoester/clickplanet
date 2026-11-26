import {ReactNode, useEffect, useState} from "react"
import {Shop} from "../../backends/shop.ts"
import {changeable, dismissable, FlagRow, flagRowsOf, Submission} from "../../domain/flagSubmissions.ts"
import {flagRenewal, flagTimeOf, payable} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {CloseIcon, FlagIcon} from "../components/icons.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import {useShop} from "../shop/shopStore.ts"
import {appealOf} from "./appeal.ts"
import {useDismissedFlagRows} from "./dismissedFlagRows.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagNote from "./FlagNote.tsx"
import {useFlags} from "./flagsStore.ts"
import SubmissionLine from "./SubmissionLine.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../support/Support.css"
import "./FlagSubmission.css"

export type FlagHubProps = {
    shop: Shop
}

type RowView = {
    name: string
    picture: ReactNode
    line?: ReactNode
    action?: ReactNode
    // Under the line: an operator's note, an appeal.
    more?: ReactNode
}

// The shop's door: a new flag, each flag the account holds with its time, and each flag it sent that did not open.
// Each row's one action opens its own page: never the plans in the list. A row that is done can be put away.
export default function FlagHub({shop}: FlagHubProps) {
    const submissions = useFlagSubmissions((store) => store.submissions)
    const held = useFlagSubmissions((store) => store.held)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const renewed = useShop((store) => store.renewed)
    const dismissed = useDismissedFlagRows((store) => store.dismissed)
    const dismiss = useDismissedFlagRows((store) => store.dismiss)
    const addNew = useFlagForm((form) => form.addNew)
    const payFor = useFlagForm((form) => form.payFor)
    const compose = useFlagForm((form) => form.compose)
    const showPaywall = useFlagForm((form) => form.showPaywall)
    const close = useFlagForm((form) => form.close)
    const [stopping, setStopping] = useState<string>()
    const [stopFailed, setStopFailed] = useState(false)
    const selling = plans.length > 0
    const rows = flagRowsOf({submissions, held}, catalog, dismissed)

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

    const button = (label: string, onClick: () => void, opens = true) =>
        <button type="button"
                className="button button-mini button-secondary flag-row-button"
                aria-haspopup={opens ? "dialog" : undefined}
                disabled={stopping !== undefined && !opens}
                onClick={onClick}>
            {label}
        </button>

    const heldView = (code: string, removalNote: string | undefined): RowView => {
        const sold = catalog.soldOf(code)
        const time = flagTimeOf(sold, catalog.closedFlag(code), flagRenewal(renewals, code, submissions))
        const line = time.kind === "removed" || selling ? flagTimeLine(time) : undefined
        const action = !selling ? undefined : time.kind === "renews"
            ? button(t("shop.stopRenewal"), () => void stop(time.renewal.id), false)
            : time.kind === "stopping" || time.kind === "open" || time.kind === "closed"
                ? button(t("shop.addTime"), () => payFor(code, true))
                : undefined
        return {
            name: translator().country(code),
            picture: <CountryFlag code={code}/>,
            line,
            action,
            more: time.kind === "removed" && removalNote ? <FlagNote text={removalNote}/> : undefined,
        }
    }

    const sentView = (submission: Submission): RowView => {
        const action = changeable(submission) && !(selling && payable(submission))
            ? button(t("flagSubmission.change"), () => compose({
                name: submission.name, pinned: submission.pin && {pin: submission.pin}, replaces: submission.id,
            }))
            : selling && payable(submission)
                ? button(t("shop.pay"), () => showPaywall(submission))
                : undefined
        const rejected = submission.state === "rejected"
        return {
            name: submission.name,
            picture: submission.picture ? <img src={submission.picture.url} alt=""/> : <FlagIcon size={20}/>,
            line: <SubmissionLine shown={submission} selling={selling}/>,
            action,
            more: rejected && <>
                {submission.rejection?.note && <FlagNote text={submission.rejection.note}/>}
                <a className="flag-row-appeal" href={appealOf(submission)}>{t("flagSubmission.appeal")}</a>
            </>,
        }
    }

    const rowOf = (row: FlagRow) => {
        const view = row.kind === "held" ? heldView(row.held.code, row.held.removalNote) : sentView(row.submission)
        const away = dismissable(row, catalog)
        return <li key={row.key} className="panel-box flag-row">
            <span className="flag-row-picture">{view.picture}</span>
            <span className="flag-row-text">
                <span className="flag-row-name">{view.name}</span>
                {view.line && <span className="flag-row-line">{view.line}</span>}
            </span>
            {view.action && <span className="flag-row-action">{view.action}</span>}
            {view.more && <div className="flag-row-more">{view.more}</div>}
            {away && <button type="button"
                             className="icon-button flag-row-dismiss"
                             aria-label={t("flagSubmission.dismiss", {flag: view.name})}
                             onClick={() => dismiss(row.key)}>
                <CloseIcon size={14}/>
            </button>}
        </li>
    }

    return <Modal title={t("flagSubmission.founded")} onClose={close}>
        <div className="flag-hub">
            <button type="button" className="button button-action support-wide" aria-haspopup="dialog" onClick={addNew}>
                <FlagIcon size={20}/>
                {t("flagSubmission.addNew")}
            </button>
            {rows.length > 0 && <ul className="flag-rows">{rows.map(rowOf)}</ul>}
            {stopFailed && <p className="support-notice" role="alert">{t("shop.stopFailed")}</p>}
        </div>
    </Modal>
}
