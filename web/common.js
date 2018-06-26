"use strict";

// https://stackoverflow.com/a/25983643
function getFormData($form) {
    var unindexed_array = $form.serializeArray();
    var indexed_array = {};

    $.map(unindexed_array, function(n, i){
        var value = n['value'];
        if (!isNaN(parseFloat(n['value']))) {
            value = parseFloat(n['value']);
        }
        indexed_array[n['name']] = value;
    });

    return indexed_array;
}

// https://stackoverflow.com/questions/10730362/get-cookie-by-name#15724300
function getCookie(name) {
    var value = "; " + document.cookie;
    var parts = value.split("; " + name + "=");
    if (parts.length == 2) return parts.pop().split(";").shift();
}

function getAuthToken() {
    if (getCookie("username") === "" || getCookie("token") === "") {
        return ""
    }
    return getCookie("username")+':'+getCookie("token");
}
