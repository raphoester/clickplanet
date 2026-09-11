export type IconProps = {
    size?: number
}

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

export function ChevronIcon({size = 18}: IconProps) {
    return <svg {...base(size)}>
        <path d="M6 9.5 12 15.5 18 9.5"/>
    </svg>
}

export function BackIcon({size = 22}: IconProps) {
    return <svg {...base(size)}>
        <path d="M14.5 5.5 8 12l6.5 6.5"/>
    </svg>
}

export function CloseIcon({size = 18}: IconProps) {
    return <svg {...base(size)}>
        <path d="M6 6 18 18"/>
        <path d="M18 6 6 18"/>
    </svg>
}

export function SwapIcon({size = 14}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.2}>
        <path d="M4 8h13l-3.5-3.5"/>
        <path d="M20 16H7l3.5 3.5"/>
    </svg>
}

export function ShareIcon({size = 14}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.2}>
        <circle cx="18" cy="5" r="2.6"/>
        <circle cx="6" cy="12" r="2.6"/>
        <circle cx="18" cy="19" r="2.6"/>
        <path d="M8.3 10.7 15.7 6.5"/>
        <path d="M8.3 13.3 15.7 17.5"/>
    </svg>
}

export function CopyIcon({size = 14}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.2}>
        <rect x="9" y="9" width="11.5" height="11.5" rx="2.6"/>
        <path d="M15 5.5A2 2 0 0 0 13 3.5H5.5a2 2 0 0 0-2 2V13a2 2 0 0 0 2 2"/>
    </svg>
}

export function DownloadIcon({size = 14}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.2}>
        <path d="M12 3.5v11"/>
        <path d="M7 10l5 4.5 5-4.5"/>
        <path d="M4 19.5h16"/>
    </svg>
}

export function SearchIcon({size = 20}: IconProps) {
    return <svg {...base(size)} strokeWidth={2} className="input-search-icon">
        <circle cx="11" cy="11" r="7"/>
        <path d="M16.5 16.5 21 21"/>
    </svg>
}
