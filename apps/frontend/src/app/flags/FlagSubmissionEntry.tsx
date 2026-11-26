import {FlagIcon} from "../components/icons.tsx"
import {matchesSearch} from "../components/visibleOptions.ts"
import {t, translator} from "../../i18n/page.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../shop/Shop.css"
import "./FlagSubmission.css"

export type FlagSubmissionEntryProps = {
    // What the picker's search typed: a flag whose time ended that it finds is claimed.
    search?: string
    // The search found no country: a new flag of that name.
    unmatched?: boolean
}

// In the flag picker, where a player looks for its region's flag: a door to a new one, or to a flag whose time ended,
// which a player claims.
export default function FlagSubmissionEntry({search = "", unmatched = false}: FlagSubmissionEntryProps) {
    const who = useFlagSubmissions((store) => store.who)
    const catalog = useFlags((store) => store.catalog)
    const open = useFlagForm((form) => form.open)
    const payFor = useFlagForm((form) => form.payFor)

    if (who === "nobody") return null

    const typed = unmatched && search.trim() !== "" ? search : undefined
    const free = search.trim() === ""
        ? undefined
        : catalog.claimableCodes()
            .map((code) => ({code, name: translator().country(code)}))
            .filter((flag) => matchesSearch(flag, search))
            .sort((a, b) => translator().compare(a.name, b.name))[0]

    if (free) {
        return <button type="button"
                       className="panel-box menu-tile flag-entry flag-entry--typed shop-shine"
                       aria-haspopup="dialog"
                       onClick={() => payFor(free.code)}>
            <FlagIcon size={20}/>
            <span className="flag-entry-name">{t("flagSubmission.claimable", {flag: free.name})}</span>
        </button>
    }

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
