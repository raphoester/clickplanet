import {FlagIcon} from "../components/icons.tsx"
import {t} from "../../i18n/page.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../shop/Shop.css"
import "./FlagSubmission.css"

export type FlagSubmissionEntryProps = {
    // What the picker's search found no country for: the flag's name.
    typed?: string
}

// In the flag picker, where a player looks for its region's flag: a door to it, as the shop's.
export default function FlagSubmissionEntry({typed}: FlagSubmissionEntryProps) {
    const who = useFlagSubmissions((store) => store.who)
    const open = useFlagForm((form) => form.open)

    if (who === "nobody") return null

    return <button type="button"
                   className={typed === undefined ? "panel-box menu-tile flag-entry shop-shine" : "panel-box menu-tile flag-entry flag-entry--typed shop-shine"}
                   aria-haspopup="dialog"
                   onClick={() => open(typed)}>
        <FlagIcon size={20}/>
        <span className="flag-entry-name">
            {typed === undefined ? t("flagSubmission.add") : t("flagSubmission.addTyped", {name: typed.trim()})}
        </span>
    </button>
}
