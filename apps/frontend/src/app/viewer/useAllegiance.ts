import {useEffect, useState} from "react";
import {AllegianceSource} from "../../backends/backend.ts";
import {Allegiance} from "../../domain/allegiance.ts";

export function useAllegiance(source: AllegianceSource | undefined): Allegiance | undefined {
    const [allegiance, setAllegiance] = useState<Allegiance>()

    useEffect(() => source?.listenForAllegiance(setAllegiance), [source])

    return allegiance
}
