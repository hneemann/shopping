let quantityModifyId = 0;

function showSetQuantity(quantity, id) {
    document.getElementById('setQuantityName').innerHTML = getNameById(id);
    quantityModifyId = id;

    let u = getUnitById(id);
    increment = u.increment;
    document.getElementById("setQuantityUnit").innerHTML = u.unit;
    document.getElementById('setQuantityQuantity').innerHTML = niceToString(quantity);
    showPopUpById('setQuantity');
}

function niceFromString(v) {
    let r = 0;
    let f = 0;
    for (let i = 0; i < v.length; i++) {
        let c = v.charAt(i);
        if (c === "¼") {
            f = 0.25;
        } else if (c === "½") {
            f = 0.5;
        } else if (c === "¾") {
            f = 0.75;
        } else {
            r = r * 10 + parseInt(c);
        }
    }
    return r + f;
}

function niceToString(v) {
    if (Number.isInteger(v)) {
        return ""+v;
    }
    let t=Math.trunc(v);
    let p=""+t
    if (p==="0") {
        p="";
    }
    switch (Math.round((v-t)*4)) {
        case 0: return p;
        case 1: return p+"¼";
        case 2: return p+"½";
        case 3: return p+"¾";
    }
    return ""+v;
}

function setQuantityMod(inc) {
    let v = niceFromString(document.getElementById('setQuantityQuantity').innerHTML);
    v += increment * inc;
    if (v <= 0) {
        v = increment;
    }
    document.getElementById('setQuantityQuantity').innerHTML = niceToString(v);
}

function setQuantityModify() {
    let text = document.getElementById('setQuantityQuantity').innerHTML;
    let v = niceFromString(text);
    updateTable("id=" + quantityModifyId + "&mode=set&q=" + v)
}

function setQuantityDelete() {
    updateTable("id=" + quantityModifyId + "&mode=del")
}

function addItem() {
    let id = document.getElementById('addItemItem').value;
    let q = niceFromString(document.getElementById('addItemQuantity').innerHTML);
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
    document.getElementById('addItemQuantity').innerHTML = niceToString(increment);
}

function modAddQuantity(inc) {
    let v = niceFromString(document.getElementById('addItemQuantity').innerText);
    v += increment * inc;
    if (v <= 0) {
        v = increment;
    }
    document.getElementById('addItemQuantity').innerText = niceToString(v);
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
