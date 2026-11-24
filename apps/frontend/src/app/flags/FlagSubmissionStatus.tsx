import {ReactNode} from "react"
import type {FlagCatalog} from "../../domain/flags.ts"
import {foundedFlags, RejectionReason, Submission, waits} from "../../domain/flagSubmissions.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {Key} from "../../i18n/translator.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {FlagIcon} from "../components/icons.tsx"
import {useFlags} from "./flagsStore.ts"
import {Draft, useFlagForm} from "./flagFormStore.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import "./FlagSubmission.css"

export type FlagSubmissionStatusProps = {
    // The newest first: the first is the one shown.
    submissions: readonly Submission[]
}

const REASONS: Record<RejectionReason, Key> = {
    "not-a-place": "flagSubmission.rejected.notAPlace",
    "exists": "flagSubmission.rejected.exists",
    "unreadable": "flagSubmission.rejected.unreadable",
    "hateful": "flagSubmission.rejected.hateful",
    "other": "flagSubmission.rejected.other",
}

function lineOf(submission: Submission): ReactNode {
    switch (submission.state) {
        case "waiting":
            return t("flagSubmission.status.waiting", {date: translator().dateTime(submission.submittedAt, {dateStyle: "medium"})})
        case "approved":
            return submission.change ? t("flagSubmission.status.changed") : t("flagSubmission.status.approved")
        case "rejected":
            return t(REASONS[submission.rejection?.reason ?? "other"])
        case "opened":
            return rich("flagSubmission.status.opened", {flag: <strong>{submission.name}</strong>})
    }
}

// The same submission again: the picture is picked again, since the page keeps none.
function againOf({name, pin, change, flag}: Submission): Draft {
    return {name, pinned: pin && {pin}, flag: change && flag !== "" ? flag : undefined}
}

function changeOf(catalog: FlagCatalog, flag: string): Draft {
    const pin = catalog.soldOf(flag)?.pin
    return {name: catalog.openedName(flag) ?? "", pinned: pin && {pin}, flag}
}

export default function FlagSubmissionStatus({submissions}: FlagSubmissionStatusProps) {
    const compose = useFlagForm((form) => form.compose)
    const close = useFlagForm((form) => form.close)
    const catalog = useFlags((store) => store.catalog)
    const [latest] = submissions
    const waiting = waits(submissions)
    const founded = foundedFlags(submissions)

    const footer = waiting
        ? <button type="button" className="button support-wide" onClick={close}>{t("common.close")}</button>
        : latest.state === "rejected"
            ? <button type="button" className="button button-action support-wide" onClick={() => compose(againOf(latest))}>{t("common.retry")}</button>
            : <button type="button" className="button button-action support-wide" onClick={() => compose()}>{t("flagSubmission.new")}</button>

    return <Modal title={t("flagSubmission.yours")} footer={footer} onClose={close}>
        <div className="flag-status">
            <div className="panel-box flag-status-card">
                <span className="flag-status-picture">
                    {latest.picture
                        ? <img src={latest.picture.url} alt=""/>
                        : latest.state === "opened" ? <CountryFlag code={latest.flag}/> : <FlagIcon size={22}/>}
                </span>
                <span className="flag-status-name">{latest.name}</span>
                <SubmissionStateChip state={latest.state}/>
            </div>
            <p className="flag-status-line" role="status">{lineOf(latest)}</p>
            {latest.state === "rejected" && latest.rejection?.note && <blockquote className="support-quoted flag-status-note">
                <span className="support-quoted-text">{latest.rejection.note}</span>
            </blockquote>}

            {founded.length > 0 && <>
                <span className="menu-label">{t("flagSubmission.founded")}</span>
                <ul className="flag-founded-list">
                    {founded.map((flag) => <li key={flag} className="panel-box flag-founded">
                        <span className="flag-founded-flag"><CountryFlag code={flag}/></span>
                        <span className="flag-founded-name">{translator().country(flag)}</span>
                        <button type="button"
                                className="button button-mini button-secondary"
                                disabled={waiting}
                                onClick={() => compose(changeOf(catalog, flag))}>
                            {t("flagSubmission.change")}
                        </button>
                    </li>)}
                </ul>
            </>}
        </div>
    </Modal>
}
