import {ReactNode, useEffect, useId, useRef} from "react";
import {BackIcon} from "./icons.tsx";
import {useEscape} from "./useDialog.ts";
import "./MenuPanel.css"

export type MenuPanelProps = {
    title: string;
    children: ReactNode;
    onClose: () => void;
}

export default function MenuPanel(props: MenuPanelProps) {
    const title = useRef<HTMLHeadingElement>(null)
    const titleId = useId()

    useEffect(() => {
        title.current?.focus()
    }, [])

    useEscape(props.onClose)

    return <section className="menu-panel" aria-labelledby={titleId}>
        <div className="menu-panel-header">
            <button type="button"
                    className="icon-button"
                    aria-label="Back"
                    onClick={props.onClose}>
                <BackIcon/>
            </button>
            <h2 className="menu-panel-title" id={titleId} ref={title} tabIndex={-1}>
                {props.title}
            </h2>
        </div>
        <div className="menu-panel-body">
            {props.children}
        </div>
    </section>
}
