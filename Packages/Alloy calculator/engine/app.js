const store = new MinecraftDataStore();
const engine = new ProductionEngine(store);
let currentResult = null;
let history = JSON.parse(localStorage.getItem('mc_history')) || [];

async function init() {
    await store.loadData(['units.json', 'materials.json', 'recipes.json']);
    
    // 1. Autofill Recipe Selector
    const recipeSelect = document.getElementById('recipe-selector');
    recipeSelect.innerHTML = '<option value="">-- Choose a Recipe --</option>';
    store.recipes.forEach(r => {
        recipeSelect.innerHTML += `<option value="${r.output}">${r.output}</option>`;
    });

    // 2. Populate Units
    const unitSelect = document.getElementById('target-unit-select');
    store.units.forEach(u => {
        unitSelect.innerHTML += `<option value="${u.id}">${u.name}</option>`;
    });

    renderInventory();
    renderHistory();
}

function autofillRecipe() {
    const selectedOutput = document.getElementById('recipe-selector').value;
    const recipe = store.getRecipe(selectedOutput);
    if (recipe) {
        document.getElementById('target-unit-select').value = recipe.unit;
    }
}

function renderInventory() {
    const container = document.getElementById('inventory-display');
    container.innerHTML = '';
    
    store.materials.forEach(mat => {
        const stock = store.inventory[mat.name] || {};
        const div = document.createElement('div');
        div.className = 'item-card';
        
        let stockRows = Object.entries(stock)
            .filter(([_, qty]) => qty > 0)
            .map(([id, qty]) => `<div>${qty}x ${id}</div>`).join('');

        div.innerHTML = `
            <h3>${mat.name}</h3>
            <div class="stock-display">${stockRows || '<em>Empty</em>'}</div>
            <div class="inventory-controls">
                <button onclick="modifyStock('${mat.name}', 'ingot', 1)">+ Ingot</button>
                <button onclick="modifyStock('${mat.name}', 'small', 1)">+ Small</button>
                <button onclick="modifyStock('${mat.name}', 'nugget', 1)">+ Nugget</button>
                <button class="btn-neg" onclick="modifyStock('${mat.name}', 'ingot', -1)">-1</button>
            </div>
        `;
        container.appendChild(div);
    });
}

function modifyStock(matName, unitId, amt) {
    store.updateInventory(matName, unitId, amt);
    renderInventory();
}

function runSolver() {
    const mat = document.getElementById('recipe-selector').value;
    const unitId = document.getElementById('target-unit-select').value;
    const amount = parseInt(document.getElementById('target-amount').value);
    
    if (!mat) return alert("Please select a recipe first.");

    const unit = store.getUnit(unitId);
    currentResult = engine.resolve(mat, unit.value * amount);
    
    // Log to History
    addToHistory(currentResult);
    
    const output = document.getElementById('solution-output');
    output.innerHTML = `<h2>Analysis: ${mat}</h2>` + formatNode(currentResult);
    
    if (currentResult.status === "Ready") {
        output.innerHTML += `<button class="btn-primary" onclick="executeCommit()">Commit Production</button>`;
    }
}

// History Logic
function addToHistory(result) {
    const entry = {
        timestamp: new Date().toLocaleTimeString(),
        target: result.material,
        status: result.status,
        needed: result.needed
    };
    history.unshift(entry);
    if (history.length > 20) history.pop(); // Keep last 20
    localStorage.setItem('mc_history', JSON.stringify(history));
    renderHistory();
}

function renderHistory() {
    const log = document.getElementById('history-log');
    if (!log) return;
    log.innerHTML = history.map(item => `
        <div class="history-item ${item.status.toLowerCase()}">
            <span class="time">${item.timestamp}</span>
            <strong>${item.target}</strong>
            <span class="status-pill">${item.status}</span>
            <small>${item.needed}ml requested</small>
        </div>
    `).join('');
}

function clearHistory() {
    history = [];
    localStorage.removeItem('mc_history');
    renderHistory();
}

// Navigation Logic
function showTab(tabId) {
    document.querySelectorAll('.tab-content').forEach(t => t.style.display = 'none');
    document.getElementById(`tab-${tabId}`).style.display = 'block';
    
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.remove('active');
        if (btn.getAttribute('onclick').includes(tabId)) btn.classList.add('active');
    });
}

window.onload = init;