import {useEffect, useMemo, useRef, useState} from "react"
import {Allegiance} from "../../domain/allegiance.ts"
import {Country} from "../../domain/countries.ts"
import {useFlags} from "../flags/flagsStore.ts"
import {Standing, stands} from "../../domain/standing.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import SelectWithSearch from "../components/SelectWithSearch.tsx"
import {useStanding} from "./standingStore.ts"
import "../CountryPicker.css"
import "./OutModal.css"

// The window opens under a player still clicking the globe, so a pick asks to be confirmed, and Confirm takes no
// click for this long after it shows: a double click on a country does not go through (the owner, 2026-11-14).
export const CONFIRM_ARM_MS = 1000

export type OutModalProps = {
    // The flag played: the one held, or a newcomer's choice.
    flag: Country
    // The flag held, if any: what the confirm step says of the next change.
    allegiance?: Allegiance
    onPick: (country: Country) => Promise<void>
}

// While the knock-out plays and the flag played is not in: the flags still in, to pick one.
export default function OutModal(props: OutModalProps) {
    const standing = useStanding((store) => store.standing)
    const watching = useStanding((store) => store.watching)

    if (standing === undefined || stands(standing, props.flag.code) || watching === props.flag.code) return null
    return <OutWindow {...props} standing={standing}/>
}

// One pick to confirm; `n` tells two picks of the same country apart, so each arms Confirm anew.
type Choice = {country: Country, n: number}

function OutWindow({flag, allegiance, onPick, standing}: OutModalProps & {standing: Standing}) {
    const watch = useStanding((store) => store.watch)
    const [choice, setChoice] = useState<Choice>()
    const [armedFor, setArmedFor] = useState<number>()
    const picks = useRef(0)
    const [sending, setSending] = useState(false)
    const [failed, setFailed] = useState(false)
    const catalog = useFlags((store) => store.catalog)
    const stillIn = useMemo(
        () => catalog.playableCodes()
            .filter((code) => stands(standing, code))
            .map((code) => ({code, name: translator().country(code)}))
            .sort((a, b) => translator().compare(a.name, b.name)),
        [standing, catalog],
    )

    useEffect(() => {
        if (choice === undefined) return
        const timer = setTimeout(() => setArmedFor(choice.n), CONFIRM_ARM_MS)
        return () => clearTimeout(timer)
    }, [choice])

    const choose = (country: Country) => {
        picks.current += 1
        setFailed(false)
        setChoice({country, n: picks.current})
    }

    const armed = choice !== undefined && armedFor === choice.n
    const confirm = async () => {
        if (choice === undefined || !armed || sending) return
        setSending(true)
        setFailed(false)
        try {
            await onPick(choice.country)
        } catch (e) {
            console.error("could not pick a flag still in", e)
            setFailed(true)
        } finally {
            setSending(false)
        }
    }

    const close = () => watch(flag.code)
    if (choice === undefined) {
        const country = translator().country(flag.code)
        return <Modal title={t(standing.out.has(flag.code) ? "knockout.isOut" : "knockout.noCapital", {country})}
                      onClose={close}>
            <p className="country-picker-rule">{t("knockout.pickStillIn")}</p>
            <div className="country-picker-list">
                <SelectWithSearch values={stillIn} onChange={(value) => choose({code: value.code})}/>
            </div>
        </Modal>
    }

    const chosen = translator().country(choice.country.code)
    return <Modal title={t("knockout.confirmTitle", {country: chosen})}
                  onClose={close}
                  footer={<div className="out-confirm-actions">
                      <button type="button" className="button button-action" disabled={!armed || sending} onClick={() => void confirm()}>
                          {t("common.confirm")}
                      </button>
                      <button type="button" className="button button-secondary" disabled={sending} onClick={() => setChoice(undefined)}>
                          {t("common.back")}
                      </button>
                  </div>}>
        <div className="out-confirm" aria-busy={sending || undefined}>
            <p className="out-confirm-flag"><CountryFlag code={choice.country.code}/><span>{chosen}</span></p>
            {allegiance?.locked && <p className="country-picker-rule">{t("knockout.keepLocked")}</p>}
            {failed && <p className="country-picker-failed" role="alert">{t("knockout.pickFailed")}</p>}
        </div>
    </Modal>
}
