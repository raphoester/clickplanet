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

export function CameraIcon({size = 20}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <path d="M3 8.8a2 2 0 0 1 2-2h2.3l1.3-2.1h6.8l1.3 2.1H19a2 2 0 0 1 2 2v8.4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
        <circle cx="12" cy="12.9" r="3.5"/>
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

export function SpeakerIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <path d="M4 9.5h3.5L12 5.5v13l-4.5-4H4z"/>
        <path d="M15.5 9a4 4 0 0 1 0 6"/>
        <path d="M18.3 6.5a7.5 7.5 0 0 1 0 11"/>
    </svg>
}

export function SpeakerOffIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <path d="M4 9.5h3.5L12 5.5v13l-4.5-4H4z"/>
        <path d="M16 9.5l5 5"/>
        <path d="M21 9.5l-5 5"/>
    </svg>
}

export function HomeIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <path d="M3.5 11 12 4l8.5 7"/>
        <path d="M6 9.5V20h4.5v-5.5h3V20H18V9.5"/>
    </svg>
}

export function InfoIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <circle cx="12" cy="12" r="8.5"/>
        <path d="M12 11v5.5"/>
        <path d="M12 7.5h.01"/>
    </svg>
}

export function DiscordIcon({size = 22}: IconProps) {
    return <svg {...base(size)} viewBox="0 0 127.14 96.36" fill="currentColor" stroke="none">
        <path d="M107.7 8.07A105.15 105.15 0 0 0 81.47 0a72.06 72.06 0 0 0-3.36 6.83 97.68 97.68 0 0 0-29.11 0A72.37 72.37 0 0 0 45.64 0a105.89 105.89 0 0 0-26.25 8.09C2.79 32.65-1.71 56.6.54 80.21a105.73 105.73 0 0 0 32.17 16.15 77.7 77.7 0 0 0 6.89-11.11 68.42 68.42 0 0 1-10.85-5.18c.91-.66 1.8-1.34 2.66-2a75.57 75.57 0 0 0 64.32 0c.87.71 1.76 1.39 2.66 2a68.68 68.68 0 0 1-10.87 5.19 77 77 0 0 0 6.89 11.1 105.25 105.25 0 0 0 32.19-16.14c2.64-27.38-4.51-51.11-18.9-72.15zM42.45 65.69C36.18 65.69 31 60 31 53s5-12.74 11.43-12.74S54 46 53.89 53s-5.05 12.69-11.44 12.69zm42.24 0C78.41 65.69 73.25 60 73.25 53s5-12.74 11.44-12.74S96.23 46 96.12 53s-5.04 12.69-11.43 12.69z"/>
    </svg>
}

export function UserIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <circle cx="12" cy="8" r="4"/>
        <path d="M4.5 20.5a7.5 7.5 0 0 1 15 0"/>
    </svg>
}

export function UsersIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <circle cx="9" cy="8.5" r="3.5"/>
        <path d="M2.5 19.5a6.5 6.5 0 0 1 13 0"/>
        <path d="M15.5 5.2a3.5 3.5 0 0 1 0 6.6"/>
        <path d="M18 14.2a6.5 6.5 0 0 1 3.5 5.3"/>
    </svg>
}

export function PlayIcon({size = 18}: IconProps) {
    return <svg {...base(size)} fill="currentColor" strokeWidth={1.6}>
        <path d="M7.5 5.2v13.6L18.5 12z"/>
    </svg>
}

export function PauseIcon({size = 18}: IconProps) {
    return <svg {...base(size)} fill="currentColor" strokeWidth={1.6}>
        <rect x="6.5" y="5.5" width="3.6" height="13" rx="0.8"/>
        <rect x="13.9" y="5.5" width="3.6" height="13" rx="0.8"/>
    </svg>
}

export function CrownIcon({size = 14}: IconProps) {
    return <svg {...base(size)} fill="currentColor" strokeWidth={1.6}>
        <path d="M3.5 8.5 8 12l4-6.5 4 6.5 4.5-3.5-2 10h-13z"/>
    </svg>
}

export function AddReactionIcon({size = 16}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.2}>
        <path d="M20.5 12.5A8.5 8.5 0 1 1 11.5 3.5"/>
        <path d="M8.5 14.5c.9 1.2 2.1 1.8 3.5 1.8s2.6-.6 3.5-1.8"/>
        <path d="M9 9.8h.01M15 9.8h.01"/>
        <path d="M18.5 2.5v6M15.5 5.5h6"/>
    </svg>
}

export function TrophyIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <path d="M8 20.5h8M12 16.5v4"/>
        <path d="M7 3.5h10v5a5 5 0 0 1-10 0z"/>
        <path d="M17 5h2.5a2 2 0 0 1 0 4H17M7 5H4.5a2 2 0 0 0 0 4H7"/>
    </svg>
}

export function ChatIcon({size = 22}: IconProps) {
    return <svg {...base(size)} strokeWidth={2}>
        <path d="M4 5h16v11H10l-5 4v-4H4z"/>
    </svg>
}

export function MoreIcon({size = 22}: IconProps) {
    return <svg {...base(size)} fill="currentColor" strokeWidth={1}>
        <circle cx="5" cy="12" r="1.8"/>
        <circle cx="12" cy="12" r="1.8"/>
        <circle cx="19" cy="12" r="1.8"/>
    </svg>
}

export function ClockIcon({size = 18}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.4}>
        <circle cx="12" cy="12" r="8.5"/>
        <path d="M12 7.5V12l3 2"/>
    </svg>
}

export function HourglassIcon({size = 12}: IconProps) {
    return <svg {...base(size)} strokeWidth={2.6}>
        <path d="M6 3h12M6 21h12M7 3v3l5 6-5 6v3M17 3v3l-5 6 5 6v3"/>
    </svg>
}
