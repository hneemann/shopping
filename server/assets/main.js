let quantityModifyId = 0;

function showSetQuantity(quantity, id) {
    document.getElementById('setQuantityName').innerHTML = getNameById(id);
    quantityModifyId = id;

    let u = getUnitById(id);
    increment = u.increment;
    document.getElementById("setQuantityUnit").innerHTML = u.unit;
    document.getElementById('setQuantityQuantity').innerHTML = "" + quantity;
    showPopUpById('setQuantity');
}

function setQuantityMod(inc) {
    let v = parseInt(document.getElementById('setQuantityQuantity').innerHTML);
    v+=increment*inc;
    if (v < increment) {
        v = increment;
    }
    document.getElementById('setQuantityQuantity').innerHTML = v;
}

function setQuantityModify() {
    let text = document.getElementById('setQuantityQuantity').innerHTML;
    let v = parseInt(text);
    updateTable("id=" + quantityModifyId + "&mode=set&q=" + v)
}

function addItem() {
    let id = document.getElementById('addItemItem').value;
    let q = document.getElementById('addItemQuantity').innerHTML;
    updateTable("id=" + id + "&mode=add&q=" + q)
}

function shopChanged() {
    updateTable("")
}

function addItemShow() {
    addItemItemChanged();
    showPopUpById('addItem')
}

function addItemCatChanged() {
    var category = document.getElementById('category').value;
    var items = document.getElementById('items').getElementsByTagName('option');
    let found = "";
    let id = "";
    for (var i = 0; i < items.length; i++) {
        if (items[i].getAttribute("data-cat") === category) {
            found += '<option value="' + items[i].id + '">' + items[i].value + '</option>';
            if (id === "") {
                id = items[i].id;
            }
        }
    }
    document.getElementById('addItemItem').innerHTML = found;
    document.getElementById('addItemLink').href = "/add?c=" + category;
    if (id !== "") {
        addItemItemChanged();
    }
}

function getNameById(id) {
    var item = document.getElementById(id);
    if (item !== null) {
        return item.value;
    }
    return "";
}

let increment = 1

function getUnitById(id) {
    let unit = "";
    let item = document.getElementById(id);
    if (item !== null) {
        unit = item.getAttribute("data-u")
    }
    let i = 1;
    if (unit === "g" || unit === "ml") {
        i = 50;
    }
    return {unit: unit, increment: i};
}

function addItemItemChanged() {
    let itemSelect = document.getElementById('addItemItem');
    let id = itemSelect.value;
    let u = getUnitById(id);
    increment = u.increment;
    document.getElementById("addItemUnit").innerHTML = u.unit;
    document.getElementById('addItemQuantity').innerHTML = "" + increment;
}

function modAddQuantity(inc) {
    let v = parseInt(document.getElementById('addItemQuantity').innerText);
    v+=increment*inc;
    if (v < increment) {
        v = increment;
    }
    document.getElementById('addItemQuantity').innerText = v;
}

let itemToDelete = -1;

function deleteItemRequest(id) {
    itemToDelete = id;
    document.getElementById('deleteNotifyName').innerHTML = getNameById(id);
    showPopUpById('deleteNotify')
}

function deleteItem() {
    hidePopUp()
    updateItem(itemToDelete, 'del')
}

function updateItem(id, mode) {
    document.getElementById(mode + "_" + id).hidden = true;
    updateTable("id=" + id + "&mode=" + mode)
}

function updateTable(query) {
    let shopElement = document.getElementById('selectedShop');
    if (shopElement !== null) {
        let shop = shopElement.value;
        if (shop !== "") {
            if (query !== "") {
                query += "&";
            }
            query += "s=" + shop;
        }
    }

    fetch("/table/?" + query, {
        signal: AbortSignal.timeout(3000)
    })
        .then(function (response) {
            if (response.status !== 200) {
                window.location.reload();
                return;
            }
            return response.text();
        })
        .catch(function (error) {
            window.location.reload();
        })
        .then(function (html) {
            let table = document.getElementById('table');
            table.innerHTML = html;
        })
}
