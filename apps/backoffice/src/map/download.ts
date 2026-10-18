export type SavedFile = {name: string, json: string}

// Hands the browser a file to save, from memory: nothing leaves the page but the save itself.
export function download({name, json}: SavedFile) {
    const url = URL.createObjectURL(new Blob([json], {type: "application/json"}))
    const link = document.createElement("a")
    link.href = url
    link.download = name
    link.click()
    // Some browsers read the link after the click returns.
    setTimeout(() => URL.revokeObjectURL(url), 10_000)
}
