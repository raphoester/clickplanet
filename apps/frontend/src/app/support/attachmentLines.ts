import {Refusal} from "../../domain/attachments.ts"
import {t} from "../../i18n/page.ts"

const KB = 1024
const MB = 1024 * KB

export function refusalLine(refusal: Refusal): string {
    switch (refusal.reason) {
        case "too-many":
            return t("support.refusal.tooMany", {most: refusal.most})
        case "too-big":
            return t("support.refusal.tooBig", {file: refusal.name, size: sizeLine(refusal.mostBytes)})
        case "not-media":
            return t("support.refusal.notMedia", {file: refusal.name})
    }
}

export function sizeLine(bytes: number): string {
    if (bytes < KB) return t("support.size.bytes", {size: bytes})
    if (bytes < MB) return t("support.size.kilobytes", {size: Math.round(bytes / KB)})

    const megabytes = bytes / MB
    return t("support.size.megabytes", {size: megabytes < 10 ? Math.round(megabytes * 10) / 10 : Math.round(megabytes)})
}
