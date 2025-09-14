function modify(id, n) {
    fetch("/listAllMod/?id=" + id + "&n=" + n)
        .then(function (response) {
            if (response.status !== 200) {
                window.location.reload();
                return;
            }
            return response.text()
        })
        .then(function (html) {
            let q = document.getElementById('q' + id);
            q.innerHTML = html;
        })
}

function saveCategories() {
    let cat = document.getElementById('categoriesInput').value;
    fetch("/listAllMod/?cat=" + encodeURIComponent(cat))
        .then(function (response) {
            window.location.reload();
        })
}