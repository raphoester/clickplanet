import {NameColor, PlayerInfo, PlayerLine} from "../../backends/player.ts"
import {t} from "../../i18n/page.ts"

export const LEAVE_GRACE_MS = 300

export function deletedLineOf(countryCode: string): PlayerLine {
    return {
        accountId: "",
        name: t("player.deletedAccount"),
        countryCode,
        guest: false,
        admin: false,
        color: NameColor.UNSPECIFIED,
    }
}

export function playerLineOf(accountId: string, countryCode: string, info: PlayerInfo): PlayerLine {
    return {
        accountId,
        name: info.name,
        countryCode,
        guest: info.guest,
        admin: info.admin,
        color: info.color,
    }
}
