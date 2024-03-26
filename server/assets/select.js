function showSetQuantity(name, quantity, id) {
    document.getElementById('setQuantityName').innerHTML = name;
    document.getElementById('setQuantitySelect').value = quantity;
    document.getElementById('setQuantityId').value = id;
    showPopUpById('setQuantity');
}

function catChanged() {
    var category = document.getElementById('category').value;
    var items = document.getElementById('items').getElementsByTagName('option');
    let found = [];
    for (var i = 0; i < items.length; i++) {
        if (items[i].id === category) {
            found.push(items[i].value)
        }
    }
    document.getElementById('selectItem').innerHTML = found.map(e => '<option value="' + e + '">' + e + '</option>').join('');
    document.getElementById('categorySelected').value = category;
}

function update(id, mode) {
    document.getElementById(mode + "_" + id).hidden = true;
    fetch("/table/?id=" + id + "&mode=" + mode)
        .then(function (response) {
            return response.text()
        })
        .then(function (html) {
            let table = document.getElementById('table');
            table.innerHTML=html;
        })
}
