/*********************************
 *  File     : index.js
 *  Purpose  : Front end logic including UI, AJAX calls to backend, and encrypting / descrypting notes
 *             Encryption / Decryption is handled here to provide a true end-to-end encryption schema
 *  Authors  : Eric Caverly
 *           : Shubham Khan (enc_message, dec_message)
 */


// Effectively ripped from https://dev.to/shubhamkhan/beginners-guide-to-aes-encryption-and-decryption-in-javascript-using-cryptojs-592
function enc_message(message, psk) {
    let key = CryptoJS.SHA256(psk).toString();

    // Generate a random Initialization Vector (IV) for security
    const iv = CryptoJS.lib.WordArray.random(16);

    // Encrypt the text using AES with CBC mode and the secret key
    const encrypted = CryptoJS.AES.encrypt(message, CryptoJS.enc.Hex.parse(key), {
        iv: iv,
        padding: CryptoJS.pad.Pkcs7,
        mode: CryptoJS.mode.CBC,
    });

    // Concatenate IV and ciphertext and encode in Base64 format
    const encryptedBase64 = CryptoJS.enc.Base64.stringify(
        iv.concat(encrypted.ciphertext)
    );

    return encryptedBase64;
}


// Effectively ripped from https://dev.to/shubhamkhan/beginners-guide-to-aes-encryption-and-decryption-in-javascript-using-cryptojs-592
function dec_message(encText, psk) {
    let key = CryptoJS.SHA256(psk).toString();

    try {
        const fullCipher = CryptoJS.enc.Base64.parse(encText);

        // Extract IV and ciphertext from the parsed cipher
        const iv = CryptoJS.lib.WordArray.create(fullCipher.words.slice(0, 4), 16);
        const ciphertext = CryptoJS.lib.WordArray.create(fullCipher.words.slice(4));

        const cipherParams = CryptoJS.lib.CipherParams.create({
            ciphertext: ciphertext,
        });

        // Decrypt the ciphertext using AES and the provided secret key
        const decrypted = CryptoJS.AES.decrypt(cipherParams, CryptoJS.enc.Hex.parse(key), {
            iv: iv,
            padding: CryptoJS.pad.Pkcs7,
            mode: CryptoJS.mode.CBC,
        });

        // Return decrypted text in UTF-8 format
        return decrypted.toString(CryptoJS.enc.Utf8);

    } catch (error) {
        console.error("decryption error:", error);
        return null;
    }
}


// Specialized wrapper for making AJAX requests to the backend
function api_req(method, endpoint, data, success_func) {
    const opt = {
        url: `/api/${endpoint}`,
        type: method,
        data: data
    }

    let req_obj = $.ajax(opt);

    req_obj.fail((xhr_err, _, err) => {
        console.log(xhr_err);
        console.log(err);
        alert(`There was a problem making the request`);
    });

    req_obj.done(success_func);
}


// Build UI for when creating a note
function setup_note_creation() {
    const form = $("#note_form");
    const loading = $("#loading_card");
    const create_note = $("#create_card");
    const result_card = $("#result_card");
    const result_body = $("#result_body");
    const new_expiry = $("#new_expiry");

    for (let i=1; i<16; ++i) {
        const opt = document.createElement("option")
        opt.value = i;
        opt.appendChild(document.createTextNode(`${i} day(s)`))
        new_expiry.append(opt);
    }

    form.submit((e) => {
        e.preventDefault();

        let msg = $("#new_content").val();
        let psk = $("#new_passphrase").val().trim();
        let ipr = $("#new_ip_restriction").val().trim();
        let limit_click_b = $("#new_limit_clicks").is(":checked");
        let max_click_i = $("#new_max_clicks").val();
        let num_links_i = $("#new_num_links").val();
        let exp = new_expiry.val();

        let ciphertext = enc_message(msg, psk);

        create_note.hide();
        loading.show();

        api_req("POST", "note", {
            "content": ciphertext,
            "allowed_ips": ipr,
            "days_until_expire": exp,
            "limit_clicks": limit_click_b,
            "max_clicks": max_click_i,
            "num_links": num_links_i,
        }, (result) => {
            loading.hide();
            result_card.show();

            if (result.success) {
                result_body.empty();

                result.data.forEach(id => {
                    const url = `${window.location.href}?uuid=${id}`;
                    const btn = document.createElement("button");
                    btn.setAttribute("class", "btn btn-primary");
                    btn.innerHTML = `&#x1F4CB;`;
                    btn.addEventListener("click", () => {
                        navigator.clipboard.writeText(url);
                    });
                    const id_sh = document.createElement("h6");
                    id_sh.style = "font-size: 11px;"
                    id_sh.appendChild(document.createTextNode(id));

                    result_body.append(`&#x2713; Note available <a class="text-secondary" href="${url}">here</a> `);
                    result_body.append(btn);
                    result_body.append(id_sh)
                    result_body.append(document.createElement("hr"));
                });

            } else {
                create_note.show();
                result_body.html(`&#x274c; Error: ${result.message}`);
            }
        });
    })

    loading.hide();
    create_note.show();
}


// Build UI when opening / decrypting a note
function setup_note_retrieval(uuid) {
    const loading = $("#loading_card");
    const dec_card = $("#decrypt_card");
    const result_card = $("#result_card");
    const result_body = $("#result_body");
    const result_back = $("#result_back");

    // Obtain the note, rendering a field for the passphrase if the note eixsts, or an error
    api_req("GET", `note/${uuid}`, {}, (result) => {
        console.log(result)
        if (result.success) {
            $("#decrypt_form").submit((e) => {
                e.preventDefault();
                loading.show();

                let psk = $("#view_passphrase").val().trim();
                let msg = dec_message(result.data.content, psk);

                result_body.empty();

                if (msg == "" || msg == null) {
                    result_body.html(`&#x274c; Invalid passphrase`);
                } else {
                    result_body.html("<h5>Note Content:</h5>")
                    const code = document.createElement("code")
                    const pre = document.createElement("pre")
                    pre.appendChild(document.createTextNode(msg));
                    code.appendChild(pre);
                    result_body.append(code);
                }

                loading.hide();
                result_back.hide();
                result_card.show();
            });

            loading.hide();
            dec_card.show();
        } else {
            loading.hide();
            result_back.show();
            result_card.show();
            result_body.html(`&#x274c; Error: ${result.message}`);
        }
    });
}


$(() => {
    // Check if the UUID is specified as a Query Parameter
    const params = new Proxy(new URLSearchParams(window.location.search), {
        get: (searchParams, prop) => searchParams.get(prop),
    });
    let uuid = params.uuid;

    // Render UI accordingly
    if (uuid == null) {
        setup_note_creation();
    } else {
        setup_note_retrieval(uuid);
    }
});