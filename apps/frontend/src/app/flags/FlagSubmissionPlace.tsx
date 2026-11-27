import {useEffect} from "react"
import {FlagSubmitter} from "../../backends/flags.ts"
import {Shop} from "../../backends/shop.ts"
import {t} from "../../i18n/page.ts"
import {AccountStore} from "../account/accountStore.ts"
import SignInButtons from "../account/SignInButtons.tsx"
import {useAccount} from "../account/useAccount.ts"
import Modal from "../components/Modal.tsx"
import {CrownIcon} from "../components/icons.tsx"
import {useEscape} from "../components/useDialog.ts"
import {useShop} from "../shop/shopStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagHub from "./FlagHub.tsx"
import FlagPaywall from "./FlagPaywall.tsx"
import FlagPayScreen from "./FlagPayScreen.tsx"
import FlagPitch from "./FlagPitch.tsx"
import FlagSubmissionForm from "./FlagSubmissionForm.tsx"
import {FlagShooter} from "./mapShot.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../account/Account.css"
import "./FlagSubmission.css"

export type FlagSubmissionPlaceProps = {
    account: AccountStore
    submitter: FlagSubmitter
    // The map at the capital with the flag on it.
    shooter: FlagShooter
    shop: Shop
    // Stripe's page.
    toCheckout: (url: string) => void
}

// The one way to a region's flag, which every door opens: the account's flags, a flag made then paid for, a flag's time
// or its claim, or a guest's sign-in. Mounted with the game, so what was typed stays while the form is put aside to aim
// the pin.
export default function FlagSubmissionPlace({account: store, submitter, shooter, shop, toCheckout}: FlagSubmissionPlaceProps) {
    const shown = useFlagForm((form) => form.shown)
    const step = useFlagForm((form) => form.step)
    const aiming = useFlagForm((form) => form.aiming)
    const land = useFlagForm((form) => form.land)
    const close = useFlagForm((form) => form.close)
    const read = useFlagSubmissions((store) => store.read)
    const account = useAccount(store)

    useEffect(() => {
        if (!shown) return
        useFlagSubmissions.getState().refresh()
        useShop.getState().refresh()
    }, [shown])

    useEffect(() => {
        if (read) land()
    }, [read, step, land])

    if (!shown) return null
    if (aiming) return <PinAim/>
    if (account.kind !== "ready") return null

    if (account.me.linked.length === 0) {
        return <Modal title={t("flagSubmission.add")} onClose={close}>
            <div className="account-panel flag-sign-in" aria-busy={account.busy !== undefined}>
                <FlagPitch/>
                <p className="account-text">{t("flagSubmission.signIn")}</p>
                <SignInButtons state={account} store={store}/>
            </div>
        </Modal>
    }

    switch (step.kind) {
        case "door":
            return <Modal title={t("flagSubmission.add")} onClose={close}>
                <p className="flag-status-line" role="status">{t("common.loading")}</p>
            </Modal>
        case "hub":
            return <FlagHub/>
        case "make":
            return <FlagSubmissionForm submitter={submitter} shooter={shooter}/>
        case "pay":
            return <FlagPaywall submission={step.submission} typed={step.typed} shooter={shooter} submitter={submitter} shop={shop}
                                toCheckout={toCheckout}/>
        case "flag":
            return <FlagPayScreen flag={step.flag} fromHub={step.fromHub} shooter={shooter} shop={shop} toCheckout={toCheckout}/>
    }
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
