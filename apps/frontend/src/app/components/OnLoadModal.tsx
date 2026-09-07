import {ReactNode, useState} from "react";
import Modal from "./Modal.tsx";

export type OnLoadModalProps = {
    title: string,
    children: ReactNode
}

export default function OnLoadModal(props: OnLoadModalProps) {
    const [isOpen, setIsOpen] = useState(true)
    if (!isOpen) return null

    return <div className="modal-onload">
        <Modal onClose={() => setIsOpen(false)} title={props.title}>
            {props.children}
        </Modal>
    </div>
}
