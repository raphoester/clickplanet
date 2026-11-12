import {endPartsOf} from "../domain/endTime.ts"
import {translator} from "../i18n/page.ts"

export function describeEnd(at: number, now: number): string {
    return translator().dateTime(at, endPartsOf(at, now))
}
