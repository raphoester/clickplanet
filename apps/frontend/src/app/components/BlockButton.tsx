export type BlockButtonProps = {
    text: string;
    imageUrl?: string;
    className?: string;
    onClick?: () => void;
}

export default function BlockButton(props: BlockButtonProps) {
    return <button
        onClick={props.onClick}
        className={`button ${props.className ?? ""}`}>
        {props.text}
    </button>
}
