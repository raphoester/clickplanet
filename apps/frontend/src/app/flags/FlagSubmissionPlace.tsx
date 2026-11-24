import {useEffect} from "react"
import {FlagSubmitter} from "../../backends/flags.ts"
import {t} from "../../i18n/page.ts"
import {AccountStore} from "../account/accountStore.ts"
import SignInButtons from "../account/SignInButtons.tsx"
import {useAccount} from "../account/useAccount.ts"
import Modal from "../components/Modal.tsx"
import {CrownIcon} from "../components/icons.tsx"
import {useEscape} from "../components/useDialog.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagSubmissionForm from "./FlagSubmissionForm.tsx"
import FlagSubmissionStatus from "./FlagSubmissionStatus.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../account/Account.css"
import "./FlagSubmission.css"

export type FlagSubmissionPlaceProps = {
    account: AccountStore
    submitter: FlagSubmitter
}

// The flag form's place: the status of the latest submission, or the form, or a guest's sign-in. Mounted with the
// game, so what was typed stays while the form is put aside to aim the pin.
export default function FlagSubmissionPlace({account: store, submitter}: FlagSubmissionPlaceProps) {
    const shown = useFlagForm((form) => form.shown)
    const composing = useFlagForm((form) => form.composing)
    const aiming = useFlagForm((form) => form.aiming)
    const close = useFlagForm((form) => form.close)
    const submissions = useFlagSubmissions((store) => store.submissions)
    const read = useFlagSubmissions((store) => store.read)
    const account = useAccount(store)

    useEffect(() => {
        if (shown) useFlagSubmissions.getState().refresh()
    }, [shown])

    if (!shown) return null
    if (aiming) return <PinAim/>
    if (account.kind !== "ready") return null

    if (account.me.linked.length === 0) {
        return <Modal title={t("flagSubmission.add")} onClose={close}>
            <div className="account-panel flag-sign-in" aria-busy={account.busy !== undefined}>
                <p className="account-text">{t("flagSubmission.signIn")}</p>
                <SignInButtons state={account} store={store}/>
            </div>
        </Modal>
    }

    if (composing || (read && submissions.length === 0)) return <FlagSubmissionForm submitter={submitter}/>
    if (!read) {
        return <Modal title={t("flagSubmission.yours")} onClose={close}>
            <p className="flag-status-line" role="status">{t("common.loading")}</p>
        </Modal>
    }
    return <FlagSubmissionStatus submissions={submissions}/>
}

// The form put aside: a tap on the globe puts the pin, Escape or Cancel brings the form back as it was.
function PinAim() {
    const putAway = useFlagForm((form) => form.putAway)
    useEscape(putAway)

    return <div className="pin-aim" role="status" aria-live="polite">
        <div className="pin-aim-line panel">
            <CrownIcon size={18}/>
            <span>{t("flagSubmission.pinAim")}</span>
            <button type="button" className="button button-mini button-secondary" onClick={putAway}>{t("common.cancel")}</button>
        </div>
    </div>
}
