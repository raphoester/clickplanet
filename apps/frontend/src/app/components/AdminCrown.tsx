import {CrownIcon} from "./icons.tsx"
import "./AdminCrown.css"

export default function AdminCrown({size = 14}: {size?: number}) {
    return <span className="admin-crown" role="img" aria-label="Admin" title="Admin">
        <CrownIcon size={size}/>
    </span>
}
