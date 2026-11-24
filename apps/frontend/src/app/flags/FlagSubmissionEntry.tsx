import {shownOf} from "../../domain/flagSubmissions.ts"
import {FlagIcon} from "../components/icons.tsx"
import {t} from "../../i18n/page.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import {latestSubmission, useFlagSubmissions} from "./submissionsStore.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import "./FlagSubmission.css"

// At the end of the flag picker: where a player looks for its region's flag.
export default function FlagSubmissionEntry() {
    const who = useFlagSubmissions((store) => store.who)
    const latest = useFlagSubmissions(latestSubmission)
    const catalog = useFlags((store) => store.catalog)
    const show = useFlagForm((form) => form.show)

    if (who === "nobody") return null

    return <button type="button" className="panel-box menu-tile flag-entry" aria-haspopup="dialog" onClick={show}>
        <FlagIcon size={20}/>
        <span className="flag-entry-name">{latest ? latest.name : t("flagSubmission.add")}</span>
        {latest && <SubmissionStateChip state={shownOf(latest, catalog).state}/>}
    </button>
}
