export type MenuButtonProps = {
    text: string;
    /** Drives `aria-expanded`: this button owns the panel below it. */
    expanded: boolean;
    onClick: () => void;
}

/** A button in the menu card that expands a MenuPanel underneath the actions. */
export default function MenuButton(props: MenuButtonProps) {
    return <button
        onClick={props.onClick}
        aria-expanded={props.expanded}
        className="button">
        {props.text}
    </button>
}
