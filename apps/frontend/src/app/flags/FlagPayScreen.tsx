import {useEffect, useState} from "react"
import {FlagSubmitter, SubmissionRefusedError} from "../../backends/flags.ts"
import {Shop} from "../../backends/shop.ts"
import {flagRenewal, flagTimeOf} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import Modal from "../components/Modal.tsx"
import {flagTimeLine} from "../shop/flagTime.ts"
import PlanOffer from "../shop/PlanOffer.tsx"
import {useShop} from "../shop/shopStore.ts"
import AnthemPicker from "./AnthemPicker.tsx"
import {coordinatesOf} from "./coordinates.ts"
import FlagCard from "./FlagCard.tsx"
import {useFlagForm} from "./flagFormStore.ts"
import FlagNote from "./FlagNote.tsx"
import {useFlags} from "./flagsStore.ts"
import {FlagShooter, useMapShot} from "./mapShot.ts"
import {SubmissionStateChip} from "./SubmissionStateChip.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"
import "../support/Support.css"
import "./FlagSubmission.css"

export type FlagPayScreenProps = {
    flag: string
    // Back goes to the account's flags.
    fromHub: boolean
    // The map at the capital with the flag on it.
    shooter: FlagShooter
    shop: Shop
    // The anthem its holder sets.
    submitter: FlagSubmitter
    // Stripe's page.
    toCheckout: (url: string) => void
}

// A flag's own page, as a new flag's paywall: the flag on the game's map at its capital, its time, then what may be done.
// Its holder adds time or stops its renewal; a player claims one whose time ended; a removed one shows why.
export default function FlagPayScreen({flag, fromHub, shooter, shop, submitter, toCheckout}: FlagPayScreenProps) {
    const catalog = useFlags((store) => store.catalog)
    const held = useFlagSubmissions((store) => store.held.find(({code}) => code === flag))
    const submissions = useFlagSubmissions((store) => store.submissions)
    const plans = useShop((store) => store.plans)
    const renewals = useShop((store) => store.renewals)
    const renewed = useShop((store) => store.renewed)
    const toHub = useFlagForm((form) => form.toHub)
    const close = useFlagForm((form) => form.close)
    const [stopping, setStopping] = useState(false)
    const [stopFailed, setStopFailed] = useState(false)
    const sold = catalog.soldOf(flag)
    const closed = catalog.closedFlag(flag)
    const shot = useMapShot(shooter, sold?.pin, sold?.picture)
    const name = translator().country(flag)
    const holds = held !== undefined
    const time = flagTimeOf(sold, closed, flagRenewal(renewals, flag, submissions))
    const line = holds ? flagTimeLine(time) : undefined
    const selling = plans.length > 0
    const renewing = holds && time.kind === "renews"
    const offered = selling && sold !== undefined && !sold.removed && (holds || closed) && !renewing
    const back = fromHub ? toHub : undefined
    // A box ticked under one line is no consent to another: the offer starts again when its line changes.
    const starts = closed ? "paid" : "added"

    const stop = async (id: string) => {
        setStopping(true)
        setStopFailed(false)
        try {
            renewed(await shop.stopRenewal(id))
        } catch (e) {
            console.error("could not stop the renewal", e)
            setStopFailed(true)
        } finally {
            setStopping(false)
        }
    }

    const title = !holds ? t("flagSubmission.claimTitle", {flag: name}) : offered ? t("shop.addTime") : t("flagSubmission.yours")
    const footer = time.kind === "renews" && holds
        ? <div className="flag-pay-buttons">
            {back && <button type="button" className="button button-secondary" onClick={back}>{t("common.back")}</button>}
            <button type="button"
                    className="button button-secondary"
                    disabled={stopping}
                    onClick={() => void stop(time.renewal.id)}>
                {t("shop.stopRenewal")}
            </button>
        </div>
        : !offered && back
            ? <button type="button" className="button button-secondary support-wide" onClick={back}>{t("common.back")}</button>
            : undefined

    return <Modal title={title} onClose={close} className={offered ? "shop-modal" : undefined} footer={footer}>
        <div className="flag-status">
            <FlagCard name={name}
                      shot={shot}
                      picture={sold?.picture.url}
                      capital={sold?.pin && coordinatesOf(sold.pin)}
                      chip={!line && <SubmissionStateChip state={sold?.removed ? "removed" : closed ? "closed" : "opened"}/>}/>
            {line && <p className="flag-status-line" role="status">{line}</p>}
            {sold?.removed && held?.removalNote && <FlagNote text={held.removalNote}/>}
            {holds && sold !== undefined && !sold.removed
                && <HeldAnthem flag={flag} anthem={sold.anthem} submitter={submitter}/>}
            {stopFailed && <p className="support-notice" role="alert">{t("shop.stopFailed")}</p>}
            {offered && <PlanOffer key={`${flag}-${starts}`}
                                   shop={shop}
                                   plans={plans}
                                   target={{kind: "flag", flag}}
                                   starts={starts}
                                   toCheckout={toCheckout}
                                   onBack={back}/>}
        </div>
    </Modal>
}

type HeldAnthemProps = {flag: string, anthem: string | undefined, submitter: FlagSubmitter}

// The holder's anthem, set at once: the flags' stream tells every page the flag with it, this one too.
function HeldAnthem({flag, anthem, submitter}: HeldAnthemProps) {
    const [picked, setPicked] = useState(anthem)
    const [setting, setSetting] = useState(false)
    const [failed, setFailed] = useState(false)
    useEffect(() => setPicked(anthem), [anthem])

    const pick = async (code: string | undefined) => {
        setPicked(code)
        setSetting(true)
        setFailed(false)
        try {
            await submitter.setAnthem(flag, code)
        } catch (e) {
            if (!(e instanceof SubmissionRefusedError)) console.error("could not set the flag's anthem", e)
            setPicked(anthem)
            setFailed(true)
        } finally {
            setSetting(false)
        }
    }

    return <div className="flag-status-anthem">
        <AnthemPicker value={picked} disabled={setting} onChange={(code) => void pick(code)}/>
        {failed && <p className="support-notice" role="alert">{t("flagSubmission.anthemFailed")}</p>}
    </div>
}
