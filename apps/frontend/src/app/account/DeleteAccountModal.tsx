import Modal from "../components/Modal.tsx"
import {Provider} from "../../backends/account.ts"
import {providerList} from "./authMessages.ts"
import {t} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import "./Account.css"

export type DeleteAccountModalProps = {
    linked: Provider[]
    busy: boolean
    onConfirm: () => void
    onClose: () => void
}

export default function DeleteAccountModal(props: DeleteAccountModalProps) {
    return <Modal title={t("account.deleteTitle")}
                  stayOnBackdropClick={props.busy}
                  footer={<div className="account-confirm">
                      <button type="button"
                              className="button"
                              disabled={props.busy}
                              onClick={props.onClose}>
                          {t("common.cancel")}
                      </button>
                      <button type="button"
                              className="button button-action"
                              disabled={props.busy}
                              onClick={props.onConfirm}>
                          {t("account.deleteConfirm")}
                      </button>
                  </div>}
                  onClose={props.onClose}>
        <div className="account-confirm-text">
            <p>{t("account.deleteSignedOut")}</p>
            <p>{t("account.deleteChangedMind")}</p>
            <p>{props.linked.length > 0
                ? t("account.deleteForGoodWithLinks", {providers: providerList(props.linked)})
                : t("account.deleteForGood")}</p>
            <p>{rich("account.deleteWhatStays", {
                privacy: (children) => <a href="/privacy" target="_blank" rel="noopener noreferrer">{children}</a>,
            })}</p>
        </div>
    </Modal>
}
