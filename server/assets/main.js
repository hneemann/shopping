let quantityModifyId = 0;

function showSetQuantity(name, quantity, id) {
    document.getElementById('setQuantityName').innerHTML = name;
    document.getElementById('setQuantitySelect').value = quantity;
    quantityModifyId = id;
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

function catChanged() {
    var category = document.getElementById('category').value;
    var items = document.getElementById('items').getElementsByTagName('option');
    let found = "";
    for (var i = 0; i < items.length; i++) {
        if (items[i].getAttribute("data-cat") === category) {
            found += '<option value="' + items[i].getAttribute("data-id") + '">' + items[i].value + '</option>'
        }
    }
    document.getElementById('addItemItem').innerHTML = found;
    document.getElementById('addItemLink').href = "/add?c=" + category;
}

let itemToDelete = -1;

function deleteItemRequest(id, name) {
    itemToDelete = id;
    document.getElementById('deleteNotifyName').innerHTML = name;
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
