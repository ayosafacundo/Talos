This documentation provides a comprehensive overview of the **"Alloy Calculator" Industrial Production Manager**, a local-first, recursive calculator designed for complex Minecraft modpack logistics (like GregTech or TerraFirmaGreg).

---

# 🛠️ "Alloy Calculator" Production Manager
### *High-Precision Logistics for Technical Minecraft*

"Alloy Calculator" is a **recursive production engine** that translates high-level crafting goals into specific physical item requirements. It respects strict integer split rules (e.g., 1 Ingot = 9 Nuggets) and prioritizes clearing "inventory clutter" before breaking down larger units.

---

## 🚀 Key Features

### 1. **Recursive Dependency Solver**
The engine doesn't just look at the top-level recipe. If you want to craft a **Bronze Ingot**, the app checks your inventory for Bronze. If you have none, it automatically calculates the requirements for **Copper** and **Tin**, traversing the entire recipe tree until it finds raw materials or missing gaps.

### 2. **Strict Integer Quantization**
Unlike standard calculators that might suggest "0.4 Ingots," "Alloy Calculator" operates on a **Fluid Value Standard** (Default: 144ml per Ingot). 
* It ensures all outputs are **Integers** of your target unit.
* It respects **Strict Split Trees**: It knows 1 Ingot can become 9 Nuggets ($16 \text{ml} \times 9 = 144\text{ml}$), but it won't suggest converting a Small Piece into a Nugget if that transition isn't defined in your `units.json`.

### 3. **Clutter-First Inventory Logic**
The solver is designed to keep your digital warehouse clean. When calculating a mix, it follows this priority:
1. **Match:** Use exact units in stock.
2. **Consolidate (Smelt):** Use smaller units (Nuggets) to reach the 144ml threshold before touching whole Ingots.
3. **Split:** Only break a whole Ingot if the recipe cannot be satisfied by smaller existing pieces.

### 4. **Strategic Pathfinding**
If an item has multiple recipes, the user can toggle between:
* **Inventory First:** Choose the path that uses the most materials currently in your `localStorage`.
* **Cheapest Fluid:** Choose the path that requires the lowest total volume of raw materials.

### 5. **Local-First & Privacy-Centric**
* **Browser Storage:** Your inventory is saved directly in your browser's `localStorage`. No cloud sync, no accounts, total data sovereignty.
* **JSON Configuration:** Add new mods, items, or custom split rules simply by editing the `units.json`, `materials.json`, and `recipes.json` files.

---

## 📂 File Architecture

The application is modularized into specialized logic layers:

| File | Responsibility |
| :--- | :--- |
| `index.html` | The UI shell containing the Workbench, Inventory, and History tabs. |
| `style.css` | Industrial Dark Theme and responsive layout. |
| `app.js` | The "Controller" that links the UI to the underlying engines. |
| `data_store.js` | Manages JSON fetching and persistent `localStorage` operations. |
| `production_engine.js` | **The Brain.** Handles recursive DFS pathfinding and integer math. |
| `units.json` | Defines physical units (Ingot, Nugget) and their ml values. |
| `recipes.json` | The knowledge base of all craftable mixes. |

---

## 📋 Data Schema Examples

### Defining a Split Tree (`units.json`)
This defines how items break down. You can have isolated trees (e.g., Ingots can become Nuggets, but Nuggets can't become Small Pieces).

```json
{
  "units": [
    {"id": "ingot", "name": "Ingot", "value": 144},
    {"id": "nugget", "name": "Nugget", "value": 16}
  ],
  "transformations": [
    {"from": "ingot", "to": "nugget", "qty": 9}
  ]
}
```

### Defining a Recipe (`recipes.json`)
Recipes use percentages. The engine calculates the required fluid based on these ratios.

```json
{
  "output": "Bronze",
  "unit": "ingot",
  "ingredients": [
    {"material": "Copper", "min": 75, "max": 75},
    {"material": "Tin", "min": 25, "max": 25}
  ]
}
```

---

## 🛠️ User Workflow

1. **Setup Inventory:** Go to the **Inventory Tab** and input your current stock of Ingots, Small Pieces, or Nuggets.
2. **Select Recipe:** In the **Workbench**, select your target output (e.g., Bronze).
3. **Analyze:** Click **Calculate**. The app will show a tree view of:
   * What can be taken from stock.
   * What needs to be crafted.
   * What is missing (highlighted in red).
4. **Commit:** Once the status is **"Ready"**, click **Commit Production**. The app will automatically subtract the ingredients and add the finished product to your Inventory.
5. **Review:** Check the **History Tab** to see a log of your latest industrial calculations.

---

## ⚠️ Requirements
* **Local Server:** Due to browser security (CORS), the JSON files must be served via a local server (e.g., `npx serve`, VS Code Live Server, or Python's `http.server`).
* **Modern Browser:** Requires ES6+ support for Classes and Async/Await.