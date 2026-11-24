import {ReactNode, useState} from "react"
import {Shop} from "../../backends/shop.ts"
import {foundedFlags, Submission} from "../../domain/flagSubmissions.ts"
import {FlagTime, flagRenewal, flagTimeOf} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import {useShop} from "../shop/shopStore.ts"
import FlagNote from "./FlagNote.tsx"
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

// The flags the account founded, and while the shop is open each one's time: Add time, or its renewal. None changes
// once approved.
function closeNoteOf(submissions: readonly Submission[], flag: string): string | undefined {
    return submissions.find((submission) => submission.state === "opened" && submission.flag === flag)?.closeNote
}

export default function FoundedFlags({submissions, shop, noteShownFor, onAddTime}: FoundedFlagsProps) {
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const renewed = useShop((store) => store.renewed)
    const [now] = useState(() => Date.now())
    const [stopping, setStopping] = useState<string>()
    const [stopFailed, setStopFailed] = useState(false)
    const founded = foundedFlags(submissions)
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
                const line = flagTimeLine(time)
                const closed = catalog.closedFlag(flag)
                const closeNote = closed && flag !== noteShownFor ? closeNoteOf(submissions, flag) : undefined
                return <li key={flag} className="panel-box flag-founded">
                    <span className="flag-founded-flag"><CountryFlag code={flag}/></span>
                    <span className="flag-founded-text">
                        <span className="flag-founded-name">{translator().country(flag)}</span>
                        {line && <span className="flag-founded-time">{line}</span>}
                    </span>
                    {closed && <SubmissionStateChip state="closed"/>}
                    {actionOf(flag, time) && <span className="flag-founded-actions">{actionOf(flag, time)}</span>}
                    {closeNote && <FlagNote text={closeNote}/>}
                </li>
            })}
        </ul>
        {stopFailed && <p className="support-notice" role="alert">{t("shop.stopFailed")}</p>}
    </>
}
