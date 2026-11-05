export type SavedFile = {name: string, json: string}

export function download({name, json}: SavedFile) {
    saveText(name, json, "application/json")
}

// Hands the browser a file to save, from memory: nothing leaves the page but the save itself.
export function saveText(name: string, text: string, type: string) {
    const url = URL.createObjectURL(new Blob([text], {type}))
    const link = document.createElement("a")
    link.href = url
    link.download = name
    link.click()
    // Some browsers read the link after the click returns.
    setTimeout(() => URL.revokeObjectURL(url), 10_000)
}
