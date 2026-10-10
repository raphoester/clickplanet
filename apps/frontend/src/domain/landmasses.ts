// The tiles of each landmass, from the borders blob's tile → landmass table.
export class Landmasses {
    private readonly starts: Uint32Array
    private readonly members: Uint32Array

    constructor(assignment: Uint16Array, count: number) {
        this.starts = new Uint32Array(count + 1)
        for (const landmass of assignment) this.starts[landmass + 1]++
        for (let i = 1; i <= count; i++) this.starts[i] += this.starts[i - 1]

        this.members = new Uint32Array(assignment.length)
        const next = this.starts.slice(0, count)
        assignment.forEach((landmass, i) => {
            this.members[next[landmass]++] = i + 1
        })
    }

    public tilesOf(landmass: number): Uint32Array {
        if (!Number.isInteger(landmass) || landmass < 0 || landmass + 1 >= this.starts.length) return new Uint32Array()
        return this.members.subarray(this.starts[landmass], this.starts[landmass + 1])
    }
}
