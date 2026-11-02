import {useId, useState} from "react"
import Modal from "../components/Modal.tsx"
import {authorStyle} from "../chat/authorStyle.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {youLabel} from "../youLabel.ts"
import {useAccount} from "./useAccount.ts"
import {AccountSettings} from "./AccountPanel.tsx"
import ProgressTab from "./ProgressTab.tsx"
import DeleteAccountModal from "./DeleteAccountModal.tsx"
import type {YouPlaceProps} from "../Menu.tsx"
import "./ProfileModal.css"

export type ProfileModalProps = YouPlaceProps & {
    onClose: () => void
}

// The desktop's "You": the player's own card, as wide as the other players' cards are tall, with the account
// beside it. A guest sees only the account side, which is the sign-in.
export default function ProfileModal({account: store, linkedMultiplier, playerInfo, listenForClicks, clips, onClose}: ProfileModalProps) {
    const account = useAccount(store)
    const words = useWordFilter()
    const [confirmingDelete, setConfirmingDelete] = useState(false)
    const accountTitle = useId()

    if (!store || account.kind !== "ready") return null

    const linked = account.me.linked.length > 0

    // One dialog at a time: the confirmation takes the profile's place, as a report takes a player card's.
    if (confirmingDelete) {
        return <DeleteAccountModal linked={account.me.linked}
                                   busy={account.busy === "deleteAccount"}
                                   onConfirm={() => void store.deleteAccount().then(() => setConfirmingDelete(false))}
                                   onClose={() => setConfirmingDelete(false)}/>
    }

    const settings = <AccountSettings state={account}
                                      store={store}
                                      linkedMultiplier={linkedMultiplier}
                                      clips={clips}
                                      onDelete={() => setConfirmingDelete(true)}/>

    if (!linked) {
        return <Modal title={youLabel(false)} className="profile-modal profile-modal--guest" onClose={onClose}>
            {settings}
        </Modal>
    }

    const name = account.username
    const title = name
        ? <span className="profile-modal-name" style={authorStyle({color: account.color, guest: false})}>{words.masked(name)}</span>
        : youLabel(true)

    return <Modal title={title} className="profile-modal" onClose={onClose}>
        <div className="profile-modal-columns">
            <section className="profile-modal-progress" aria-label="Progress">
                <ProgressTab store={store}
                             me={account.me}
                             stats={playerInfo && name && account.accountId
                                 ? {backend: playerInfo, accountId: account.accountId}
                                 : undefined}
                             listenForClicks={listenForClicks}/>
            </section>
            <section className="panel-box profile-modal-account" aria-labelledby={accountTitle}>
                <h3 className="menu-section-title" id={accountTitle}>Account</h3>
                {settings}
            </section>
        </div>
    </Modal>
}
