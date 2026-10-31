import {useState} from "react"
import {AllegianceLockedError} from "../../backends/backend.ts"
import {Allegiance, Lock, lockOf} from "../../domain/allegiance.ts"
import {Countries, Country} from "../../domain/countries.ts"
import {describeEnd} from "../../domain/endTime.ts"
import {LOCKED_RULE} from "../CountryPicker.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
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
    const kept = Countries.get(held.countryId)?.name ?? held.countryId.toUpperCase()

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

    return <Modal title="Your friend's flag"
                  onClose={onClose}
                  footer={<div className="invited-actions">
                      <button type="button" className="button button-action" disabled={lock !== undefined || sending} onClick={() => void switchFlag()}>
                          Play for {friend.name}
                      </button>
                      <button type="button" className="button button-secondary" disabled={sending} onClick={onClose}>
                          Keep {kept}
                      </button>
                  </div>}>
        <div className="invited">
            <p className="invited-friend">
                <CountryFlag code={friend.code}/>
                <span>Your friend plays for {friend.name}.</span>
            </p>
            {lock && <p>{lockLineOf(lock, now)}</p>}
            {failed && <p role="alert">Could not change your flag. Try again.</p>}
        </div>
    </Modal>
}

function lockLineOf(lock: Lock, now: number): string {
    switch (lock.kind) {
        case "rules":
            return LOCKED_RULE
        case "wait":
            return `You can switch at ${describeEnd(lock.until, now)}.`
    }
}
