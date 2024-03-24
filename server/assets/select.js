function showSetQuantity(name,quantity,id) {
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
    document.getElementById('categorySelected').value=category;
}
