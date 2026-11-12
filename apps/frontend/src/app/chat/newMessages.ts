import {t} from "../../i18n/page.ts"

export function newMessages(unread: number): string {
    return t("chat.newMessages", {count: unread})
}
