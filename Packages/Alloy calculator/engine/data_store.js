class MinecraftDataStore {
    constructor() {
        this.inventory = JSON.parse(localStorage.getItem('mc_inventory')) || {};
        this.units = [];
        this.transformations = [];
        this.materials = [];
        this.recipes = [];
    }

    async loadData(urls) {
        try {
            const [u, m, r] = await Promise.all(urls.map(url => fetch(url).then(res => res.json())));
            this.units = u.units;
            this.transformations = u.transformations;
            this.materials = m;
            this.recipes = r;
            console.log("Data Loaded Successfully");
        } catch (e) {
            console.error("Failed to load JSON files. Ensure you are using a local server.", e);
        }
    }

    getUnit(id) { return this.units.find(u => u.id === id); }
    getRecipe(matName) { return this.recipes.find(r => r.output === matName); }

    updateInventory(material, unitId, change) {
        if (!this.inventory[material]) this.inventory[material] = {};
        const current = this.inventory[material][unitId] || 0;
        this.inventory[material][unitId] = Math.max(0, current + change);
        this.persist();
    }

    persist() {
        localStorage.setItem('mc_inventory', JSON.stringify(this.inventory));
    }
}