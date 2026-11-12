import {useState} from "react"
import {AllegianceLockedError} from "../../backends/backend.ts"
import {Allegiance, Lock, lockOf} from "../../domain/allegiance.ts"
import {Country} from "../../domain/countries.ts"
import {describeEnd} from "../endTime.ts"
import {LOCKED_LINE} from "../CountryPicker.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {t, translator} from "../../i18n/page.ts"
import "./InvitedModal.css"

export type InvitedModalProps = {
    friend: Country
    held: Allegiance
    onSwitch: () => Promise<void>
    onClose: () => void
}

export default function InvitedModal({friend, held, onSwitch, onClose}: InvitedModalProps) {
    const [sending, setSending] = useState(false)
    const [failed, setFailed] = useState(false)
    const now = Date.now()
    const lock = lockOf(held, now)
    const friendName = translator().country(friend.code)
    const kept = translator().country(held.countryId)

    const switchFlag = async () => {
        setSending(true)
        setFailed(false)
        try {
            await onSwitch()
            onClose()
        } catch (e) {
            if (e instanceof AllegianceLockedError) {
                onClose()
                return
            }
            console.error("could not play for the friend's flag", e)
            setFailed(true)
        } finally {
            setSending(false)
        }
    }

    return <Modal title={t("crazyGames.friendsFlag")}
                  onClose={onClose}
                  footer={<div className="invited-actions">
                      <button type="button" className="button button-action" disabled={lock !== undefined || sending} onClick={() => void switchFlag()}>
                          {t("crazyGames.playFor", {country: friendName})}
                      </button>
                      <button type="button" className="button button-secondary" disabled={sending} onClick={onClose}>
                          {t("crazyGames.keep", {country: kept})}
                      </button>
                  </div>}>
        <div className="invited">
            <p className="invited-friend">
                <CountryFlag code={friend.code}/>
                <span>{t("crazyGames.friendPlaysFor", {country: friendName})}</span>
            </p>
            {lock && <p>{lockLineOf(lock, now)}</p>}
            {failed && <p role="alert">{t("crazyGames.switchFailed")}</p>}
        </div>
    </Modal>
}

function lockLineOf(lock: Lock, now: number): string {
    switch (lock.kind) {
        case "rules":
            return t(LOCKED_LINE)
        case "wait":
            return t("crazyGames.switchAt", {time: describeEnd(lock.until, now)})
    }
}
