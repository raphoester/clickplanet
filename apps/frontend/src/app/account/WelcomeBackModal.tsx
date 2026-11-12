import Modal from "../components/Modal.tsx"
import {t} from "../../i18n/page.ts"
import "./Account.css"

export type WelcomeBackModalProps = {
    onClose: () => void
}

// A sign-in brought back an account its player had asked to delete: it is no longer on its way out.
export default function WelcomeBackModal(props: WelcomeBackModalProps) {
    return <Modal
        title={t("account.welcomeBack")}
        footer={<button type="button" className="button button-action" onClick={props.onClose}>{t("account.welcomeBackPlay")}</button>}
        onClose={props.onClose}>
        <div className="account-confirm-text">
            <p>{t("account.welcomeBackText")}</p>
        </div>
    </Modal>
}
