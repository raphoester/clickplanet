import {CrownIcon} from "./icons.tsx"
import {t} from "../../i18n/page.ts"
import "./AdminCrown.css"

export default function AdminCrown({size = 14}: {size?: number}) {
    const admin = t("board.admin")
    return <span className="admin-crown" role="img" aria-label={admin} title={admin}>
        <CrownIcon size={size}/>
    </span>
}
