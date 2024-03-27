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
        if (items[i].id === category) {
            found+='<option value="' + i + '">' + items[i].value + '</option>'
        }
    }
    document.getElementById('addItemItem').innerHTML = found;
}

function updateItem(id, mode) {
    document.getElementById(mode + "_" + id).hidden = true;
    updateTable("id=" + id + "&mode=" + mode)
}

function updateTable(query) {
    fetch("/table/?" + query)
        .then(function (response) {
            return response.text()
        })
        .then(function (html) {
            let table = document.getElementById('table');
            table.innerHTML = html;
        })
}
