import {ReactNode, useEffect} from "react"
import {dismissable, FlagRow, flagRowsOf} from "../../domain/flagSubmissions.ts"
import {flagRenewal, flagTimeOf} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {ChevronIcon, CloseIcon, FlagIcon} from "../components/icons.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import {useShop} from "../shop/shopStore.ts"
import {useDismissedFlagRows} from "./dismissedFlagRows.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import SubmissionLine from "./SubmissionLine.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "./FlagSubmission.css"

type RowView = {name: string, picture: ReactNode, line?: ReactNode, open: () => void}

// The shop's door: each flag the account holds and each flag it sent that did not open, one line each. A row opens the
// flag's own page, where its plans, its renewal or its verdict are; a row that is done can be put away.
export default function FlagHub() {
    const submissions = useFlagSubmissions((store) => store.submissions)
    const held = useFlagSubmissions((store) => store.held)
    const catalog = useFlags((store) => store.catalog)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const dismissed = useDismissedFlagRows((store) => store.dismissed)
    const dismiss = useDismissedFlagRows((store) => store.dismiss)
    const addNew = useFlagForm((form) => form.addNew)
    const payFor = useFlagForm((form) => form.payFor)
    const showPaywall = useFlagForm((form) => form.showPaywall)
    const close = useFlagForm((form) => form.close)
    const selling = plans.length > 0
    const rows = flagRowsOf({submissions, held}, catalog, dismissed)

    useEffect(() => {
        useFlagSubmissions.getState().refresh()
        useShop.getState().refresh()
    }, [])

    const viewOf = (row: FlagRow): RowView => {
        if (row.kind === "held") {
            const {code} = row.held
            const sold = catalog.soldOf(code)
            const time = flagTimeOf(sold, catalog.closedFlag(code), flagRenewal(renewals, code, submissions))
            return {
                name: translator().country(code),
                picture: sold ? <img src={sold.picture.url} alt=""/> : <CountryFlag code={code}/>,
                line: time.kind === "removed" || selling ? flagTimeLine(time) : undefined,
                open: () => payFor(code, true),
            }
        }
        const {submission} = row
        return {
            name: submission.name,
            picture: submission.picture ? <img src={submission.picture.url} alt=""/> : <FlagIcon size={18}/>,
            line: <SubmissionLine shown={submission} selling={selling}/>,
            open: () => showPaywall(submission),
        }
    }

    const rowOf = (row: FlagRow) => {
        const view = viewOf(row)
        const away = dismissable(row, catalog)
        return <li key={row.key} className="panel-box flag-row">
            <button type="button" className="flag-row-open" aria-haspopup="dialog" onClick={view.open}>
                <span className="flag-row-thumb">{view.picture}</span>
                <span className="flag-row-text">
                    <span className="flag-row-name">{view.name}</span>
                    {view.line && <span className="flag-row-line">{view.line}</span>}
                </span>
                {away && <span className="flag-row-slot"/>}
                <span className="flag-row-chevron"><ChevronIcon size={18}/></span>
            </button>
            {away && <button type="button"
                             className="flag-row-dismiss"
                             aria-label={t("flagSubmission.dismiss", {flag: view.name})}
                             onClick={() => dismiss(row.key)}>
                <CloseIcon size={16}/>
            </button>}
        </li>
    }

    return <Modal title={t("flagSubmission.founded")}
                  onClose={close}
                  className="flag-hub-modal"
                  footer={<button type="button" className="button button-action support-wide" aria-haspopup="dialog" onClick={addNew}>
                      <FlagIcon size={20}/>
                      {t("flagSubmission.addNew")}
                  </button>}>
        <ul className="flag-rows">{rows.map(rowOf)}</ul>
    </Modal>
}
