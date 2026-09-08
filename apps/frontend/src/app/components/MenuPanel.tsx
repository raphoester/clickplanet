import {ReactNode, useEffect, useId, useRef} from "react";
import CloseButton from "./CloseButton.tsx";
import {useEscape} from "./useDialog.ts";
import "./MenuPanel.css"

export type MenuPanelProps = {
    title: string;
    children: ReactNode;
    onClose: () => void;
    closeButtonText?: string;
}

/**
 * A panel that takes over the menu card's content slot, replacing the
 * leaderboard and the buttons that opened it.
 *
 * Unlike Modal it has no backdrop and no layer of its own: it blocks nothing,
 * so it is a labelled region rather than a dialog, and it neither traps the
 * keyboard nor claims to be modal. The globe stays visible and clickable.
 */
export default function MenuPanel(props: MenuPanelProps) {
    const title = useRef<HTMLHeadingElement>(null)
    const titleId = useId()

    /**
     * The button that opened this panel was unmounted in the same commit that
     * mounted it, dropping focus onto the body: move it to the heading so the
     * new context is announced. Walking back up is Menu's job — the button to
     * return to only exists again once this panel is gone.
     */
    useEffect(() => {
        title.current?.focus()
    }, [])

    useEscape(props.onClose)

    return <section className="menu-panel" aria-labelledby={titleId}>
        <h2 className="menu-panel-title" id={titleId} ref={title} tabIndex={-1}>
            {props.title}
        </h2>
        <div className="menu-panel-body">
            {props.children}
        </div>
        <CloseButton onClick={props.onClose} text={props.closeButtonText}/>
    </section>
}
