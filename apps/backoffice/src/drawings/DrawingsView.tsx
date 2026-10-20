import {useState} from "react"
import {Clients, CLIENTS} from "../clients.ts"
import {PlaceReport} from "../gen/grpc/support/v1/admin_pb.ts"
import {DrawingDetail} from "./DrawingDetail.tsx"
import {DrawingList} from "./DrawingList.tsx"

// The list is read again on the way back: a report closed in the detail leaves the open list.
export function DrawingsView({clients = CLIENTS}: {clients?: Clients}) {
    const [openOnly, setOpenOnly] = useState(true)
    const [opened, setOpened] = useState<PlaceReport | undefined>(undefined)

    if (opened !== undefined) {
        return <DrawingDetail report={opened} clients={clients} onBack={() => setOpened(undefined)}/>
    }
    return <DrawingList support={clients.support} openOnly={openOnly} onOpenOnly={setOpenOnly} onOpen={setOpened}/>
}
