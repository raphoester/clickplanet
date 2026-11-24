import {ReactNode, useState} from "react"
import {Shop} from "../../backends/shop.ts"
import type {FlagCatalog} from "../../domain/flags.ts"
import {foundedFlags, Submission, waits} from "../../domain/flagSubmissions.ts"
import {FlagTime, flagRenewal, flagTimeOf} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {useShop} from "../shop/shopStore.ts"
import FlagNote from "./FlagNote.tsx"
import {Draft, useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import "../support/Support.css"
import "./FlagSubmission.css"

export type FoundedFlagsProps = {
    // The newest first.
    submissions: readonly Submission[]
    shop: Shop
    // The closed flag whose note the place shows above, not repeated in its row.
    noteShownFor?: string
    onAddTime: (flag: string) => void
}

function changeOf(catalog: FlagCatalog, flag: string): Draft {
    const pin = catalog.soldOf(flag)?.pin
    return {name: catalog.openedName(flag) ?? "", pinned: pin && {pin}, flag}
}

function dateOf(at: number): string {
    return translator().dateTime(at, {dateStyle: "medium"})
}

function lineOf(time: FlagTime): string | undefined {
    switch (time.kind) {
        case "renews":
            return t("shop.renewsOn", {date: dateOf(time.renewal.periodEndsAt)})
        case "stopping":
            return t("shop.endsOn", {date: dateOf(time.renewal.periodEndsAt)})
        case "open":
            return t("shop.endsOn", {date: dateOf(time.endsAt)})
        case "closed":
            return t("shop.closedOn", {date: dateOf(time.closedAt)})
        case "none":
            return undefined
    }
}

// The flags the account founded, each with Change, and while the shop is open its time: Add time, or its renewal.
function closeNoteOf(submissions: readonly Submission[], flag: string): string | undefined {
    return submissions.find((submission) => submission.state === "opened" && submission.flag === flag)?.closeNote
}

export default function FoundedFlags({submissions, shop, noteShownFor, onAddTime}: FoundedFlagsProps) {
    const compose = useFlagForm((form) => form.compose)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const renewed = useShop((store) => store.renewed)
    const [now] = useState(() => Date.now())
    const [stopping, setStopping] = useState<string>()
    const [stopFailed, setStopFailed] = useState(false)
    const founded = foundedFlags(submissions)
    const waiting = waits(submissions)
    const selling = plans.length > 0

    if (founded.length === 0) return null

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

    const actionOf = (flag: string, time: FlagTime): ReactNode => {
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
                return <button type="button" className="button button-mini button-secondary" onClick={() => onAddTime(flag)}>
                    {t("shop.addTime")}
                </button>
            case "none":
                return undefined
        }
    }

    return <>
        <span className="menu-label">{t("flagSubmission.founded")}</span>
        <ul className="flag-founded-list">
            {founded.map((flag) => {
                const time: FlagTime = selling
                    ? flagTimeOf(catalog.soldOf(flag), catalog.closedFlag(flag), flagRenewal(renewals, flag, submissions), now)
                    : {kind: "none"}
                const line = lineOf(time)
                const closed = catalog.closedFlag(flag)
                const closeNote = closed && flag !== noteShownFor ? closeNoteOf(submissions, flag) : undefined
                return <li key={flag} className="panel-box flag-founded">
                    <span className="flag-founded-flag"><CountryFlag code={flag}/></span>
                    <span className="flag-founded-text">
                        <span className="flag-founded-name">{translator().country(flag)}</span>
                        {line && <span className="flag-founded-time">{line}</span>}
                    </span>
                    {closed && <SubmissionStateChip state="closed"/>}
                    <span className="flag-founded-actions">
                        <button type="button"
                                className="button button-mini button-secondary"
                                disabled={waiting}
                                onClick={() => compose(changeOf(catalog, flag))}>
                            {t("flagSubmission.change")}
                        </button>
                        {actionOf(flag, time)}
                    </span>
                    {closeNote && <FlagNote text={closeNote}/>}
                </li>
            })}
        </ul>
        {stopFailed && <p className="support-notice" role="alert">{t("shop.stopFailed")}</p>}
    </>
}
