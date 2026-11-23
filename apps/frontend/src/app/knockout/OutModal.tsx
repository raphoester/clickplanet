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
    const watch = useStanding((store) => store.watch)

    if (standing === undefined || stands(standing, props.flag.code) || watching === props.flag.code) return null
    const country = translator().country(props.flag.code)
    return <PickAnotherWindow title={t(standing.out.has(props.flag.code) ? "knockout.isOut" : "knockout.noCapital", {country})}
                              line={t("knockout.pickStillIn")}
                              standing={standing}
                              allegiance={props.allegiance}
                              onPick={props.onPick}
                              onClose={() => watch(props.flag.code)}/>
}

// One pick to confirm; `n` tells two picks of the same country apart, so each arms Confirm anew.
type Choice = {country: Country, n: number}

export type PickAnotherProps = {
    title: string
    line: string
    standing: Standing | undefined
    allegiance?: Allegiance
    onPick: (country: Country) => Promise<void>
    // None: the window stays until a flag is picked.
    onClose?: () => void
}

// The flags a player may play for, to pick one, and the pick to confirm.
export function PickAnotherWindow({title, line, standing, allegiance, onPick, onClose}: PickAnotherProps) {
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
            console.error("could not pick another flag", e)
            setFailed(true)
        } finally {
            setSending(false)
        }
    }

    if (choice === undefined) {
        return <Modal title={title} onClose={onClose} stayOnBackdropClick={onClose === undefined}>
            <p className="country-picker-rule">{line}</p>
            <div className="country-picker-list">
                <SelectWithSearch values={stillIn} onChange={(value) => choose({code: value.code})}/>
            </div>
        </Modal>
    }

    const chosen = translator().country(choice.country.code)
    return <Modal title={t("knockout.confirmTitle", {country: chosen})}
                  onClose={onClose}
                  stayOnBackdropClick={onClose === undefined}
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
