export type IconProps = {
    size?: number
}

/**
 * The menu's three navigation gestures, drawn rather than typed: emoji and
 * dingbats render differently on every platform and cannot take the button's
 * colour. Each is decorative — the button around it carries the label.
 */
const base = (size: number) => ({
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 2.4,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
})

/** Folds the whole card. Points down when open, so it flips to point up. */
export function ChevronIcon({size = 18}: IconProps) {
    return <svg {...base(size)}>
        <path d="M6 9.5 12 15.5 18 9.5"/>
    </svg>
}

/** Walks back out of the one panel you can drill into. */
export function BackIcon({size = 22}: IconProps) {
    return <svg {...base(size)}>
        <path d="M14.5 5.5 8 12l6.5 6.5"/>
    </svg>
}

/** Dismisses a dialog that opened over everything. */
export function CloseIcon({size = 18}: IconProps) {
    return <svg {...base(size)}>
        <path d="M6 6 18 18"/>
        <path d="M18 6 6 18"/>
    </svg>
}

/** Marks the control that swaps the country you play for. */
export function SwapIcon({size = 14}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.2}>
        <path d="M4 8h13l-3.5-3.5"/>
        <path d="M20 16H7l3.5 3.5"/>
    </svg>
}

/** Sits inside the search field, where a placeholder emoji used to. */
export function SearchIcon({size = 20}: IconProps) {
    return <svg {...base(size)} strokeWidth={2} className="input-search-icon">
        <circle cx="11" cy="11" r="7"/>
        <path d="M16.5 16.5 21 21"/>
    </svg>
}
