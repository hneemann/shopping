function modify(id, n) {
    fetch("/listAllMod/?id=" + id + "&n=" + n)
        .then(function (response) {
            return response.text()
        })
        .then(function (html) {
            let q = document.getElementById('q' + id);
            q.innerHTML = html;
        })
}