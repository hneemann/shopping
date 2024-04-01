let quantityModifyId = 0;

function showSetQuantity(quantity, id) {
    document.getElementById('setQuantityName').innerHTML = getNameById(id);
    document.getElementById('setQuantitySelect').value = quantity;
    quantityModifyId = id;
    setQuantitySelectId("" + id, 'setQuantitySelect', quantity);
    showPopUpById('setQuantity');
}

function modifyQuantity() {
    let q = document.getElementById('setQuantitySelect').value;
    updateTable("id=" + quantityModifyId + "&mode=set&q=" + q)
}

function addItem() {
    let id = document.getElementById('addItemItem').value;
    let q = document.getElementById('addItemQuantity').value;
    updateTable("id=" + id + "&mode=add&q=" + q)
}

function shopChanged() {
    updateTable("")
}

function catChanged() {
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
        setQuantitySelectId(id, 'addItemQuantity');
    }
}

function getNameById(id) {
    var item = document.getElementById(id);
    if (item !== null) {
        return item.value;
    }
    return "";
}

let factors = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 15, 20];

function setQuantitySelect(idItem, idQuantity) {
    let item = document.getElementById(idItem);
    let id = item.value;
    setQuantitySelectId(id, idQuantity, 1);
}

function setQuantitySelectId(id, idQuantity, quantity) {
    let unit = "";
    var item = document.getElementById(id);
    if (item !== null) {
        unit = item.getAttribute("data-u")
    }
    let multiply = 1;
    if (unit === "g" || unit === "ml") {
        multiply = 50;
    }
    let options = "";
    for (var i = 0; i < factors.length; i++) {
        let v = factors[i] * multiply;
        if (quantity == v) {
            options += '<option value="' + v + '" selected>' + v + '</option>';
        } else {
            options += '<option value="' + v + '">' + v + '</option>';
        }
    }
    document.getElementById(idQuantity + "Unit").innerHTML = unit;
    document.getElementById(idQuantity).innerHTML = options;
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
            query += "&s=" + shop;
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
