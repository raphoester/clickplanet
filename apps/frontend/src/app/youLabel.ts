import {t} from "../i18n/page.ts"

export function youLabel(linked: boolean): string {
    return linked ? t("menu.you") : t("common.signIn")
}
