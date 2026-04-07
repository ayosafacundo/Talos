class ProductionEngine {
    constructor(store) {
        this.store = store;
    }

    resolve(materialName, fluidNeeded, strategy = 'INVENTORY_FIRST') {
        let node = {
            material: materialName,
            needed: fluidNeeded,
            fromStock: {},
            toConvert: [],
            toCraft: null,
            missing: 0,
            status: "Ready"
        };

        let remaining = fluidNeeded;
        const stock = this.store.inventory[materialName] || {};

        // 1. Clutter-First: Use small units before splitting big ones
        const sortedUnits = [...this.store.units].sort((a, b) => a.value - b.value);
        
        for (const unit of sortedUnits) {
            const count = stock[unit.id] || 0;
            const canUse = Math.min(count, Math.floor(remaining / unit.value));
            if (canUse > 0) {
                node.fromStock[unit.id] = canUse;
                remaining -= (canUse * unit.value);
            }
        }

        // 2. Strict Recursive Crafting
        if (remaining > 0) {
            const recipe = this.store.getRecipe(materialName);
            if (recipe) {
                const outputUnit = this.store.getUnit(recipe.unit);
                const qtyToMake = Math.ceil(remaining / outputUnit.value);
                
                node.toCraft = {
                    unit: recipe.unit,
                    count: qtyToMake,
                    ingredients: recipe.ingredients.map(ing => {
                        const req = (qtyToMake * outputUnit.value) * (ing.min / 100);
                        return this.resolve(ing.material, req, strategy);
                    })
                };

                // Check if sub-crafts are missing items
                if (node.toCraft.ingredients.some(i => i.status === "Missing")) {
                    node.status = "Missing";
                }
            } else {
                node.missing = remaining;
                node.status = "Missing";
            }
        }

        return node;
    }

    commit(node) {
        // Remove items from stock
        for (const [unitId, qty] of Object.entries(node.fromStock)) {
            this.store.updateInventory(node.material, unitId, -qty);
        }

        // Recursively commit sub-crafts and add output
        if (node.toCraft) {
            node.toCraft.ingredients.forEach(ing => this.commit(ing));
            this.store.updateInventory(node.material, node.toCraft.unit, node.toCraft.count);
        }
        this.store.persist();
    }
}