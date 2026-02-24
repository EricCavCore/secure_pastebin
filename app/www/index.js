/*********************************
 *  File     : index.js
 *  Purpose  : Front end logic including UI, AJAX calls to backend, and encrypting / descrypting notes
 *             Encryption / Decryption is handled here to provide a true end-to-end encryption schema
 *  Authors  : Eric Caverly
 */


// Generate random salt (16 bytes recommended)
function generateSalt() {
    return window.crypto.getRandomValues(new Uint8Array(16));
}


// Derive AES key from passphrase + salt using PBKDF2
async function deriveKey(passphrase, salt, iterations = 300000) {
    const enc = new TextEncoder();
    const passKey = await window.crypto.subtle.importKey(
        'raw',
        enc.encode(passphrase),
        { name: 'PBKDF2' },
        false,
        ['deriveKey']
    );

    return await window.crypto.subtle.deriveKey(
        {
            name: 'PBKDF2',
            salt: salt,
            iterations: iterations,
            hash: 'SHA-256'
        },
        passKey,
        { name: 'AES-GCM', length: 256 },
        false,
        ['encrypt', 'decrypt']
    );
}


function uint8ToBase64(uint8Array) {
    let binary = '';
    const bytes = new Uint8Array(uint8Array);
    for (let i = 0; i < bytes.byteLength; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
}


function base64ToUint8(base64String) {
    const binary = atob(base64String);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}


async function enc_message(message, psk) {
    const msg = new TextEncoder().encode(message);
    const salt = generateSalt()

    const iv = window.crypto.getRandomValues(new Uint8Array(12));

    const key = await deriveKey(psk, salt);

    const ctBuffer = await window.crypto.subtle.encrypt({name: "AES-GCM", iv}, key, msg);

    return {
        cipher_text: uint8ToBase64(ctBuffer),
        salt: uint8ToBase64(salt),
        iv: uint8ToBase64(iv)
    };
}


async function dec_message(enc_text, enc_iv, enc_salt, psk) {
    try {
        const iv = base64ToUint8(enc_iv);
        const salt = base64ToUint8(enc_salt);
        const ciphertext = base64ToUint8(enc_text);

        const key = await deriveKey(psk, salt);

        let ptBuffer = await window.crypto.subtle.decrypt({ name: "AES-GCM", iv: iv}, key, ciphertext);

        return new TextDecoder().decode(ptBuffer);
    } catch (error) {
        console.log("Decryption failure: ", error);
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

    const passwd_field = $("#new_passphrase");
    $("#generate_password_btn").click(() => {
        const arr = new Uint32Array(3);
        window.crypto.getRandomValues(arr);

        let passphrase_words = [];
        for(let i=0; i<arr.length; ++i) {
            let index = arr[i] % word_list_length();
            passphrase_words[i] = get_word(index);
        }

        passwd_field.val(passphrase_words.join("-"));
    });

    form.submit(async (e) => {
        e.preventDefault();

        let msg = $("#new_content").val();
        let psk = passwd_field.val().trim();
        let ipr = $("#new_ip_restriction").val().trim();
        let limit_click_b = $("#new_limit_clicks").is(":checked");
        let max_click_i = $("#new_max_clicks").val();
        let num_links_i = $("#new_num_links").val();
        let exp = new_expiry.val();

        let enc_msg = await enc_message(msg, psk);

        create_note.hide();
        loading.show();

        let api_data = {
            "content": enc_msg.cipher_text,
            "iv": enc_msg.iv,
            "salt": enc_msg.salt,
            "allowed_ips": ipr,
            "days_until_expire": exp,
            "limit_clicks": limit_click_b,
            "max_clicks": max_click_i,
            "num_links": num_links_i,
        };

        api_req("POST", "note", api_data, (result) => {
            loading.hide();
            result_card.show();

            if (result.success) {
                result_body.empty();

                result.data.forEach(id => {
                    const url = `${window.location.href}?uuid=${id}`;
                    const btn = document.createElement("button");
                    btn.setAttribute("class", "btn btn-success");
                    btn.innerHTML = `&#x1F4CB;`;
                    btn.addEventListener("click", () => {
                        navigator.clipboard.writeText(url);
                    });
                    const id_sh = document.createElement("h6");
                    id_sh.style = "font-size: 11px;"
                    id_sh.appendChild(document.createTextNode(id));

                    result_body.text(` Note available`);
                    const link = document.createElement("a");
                    link.href = url;
                    link.appendChild(document.createTextNode("here"));
                    result_body.append(link);

                    result_body.append(btn);
                    result_body.append(id_sh)
                    result_body.append(document.createElement("hr"));
                });

            } else {
                create_note.show();
                result_body.text(`❌ Error: ${result.message}`);
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
        if (result.success) {
            $("#decrypt_form").submit(async (e) => {
                e.preventDefault();
                loading.show();

                let psk = $("#view_passphrase").val().trim();
                let msg = await dec_message(result.data.content, result.data.iv, result.data.salt, psk);

                result_body.empty();

                if (msg == "" || msg == null) {
                    result_body.text(`❌ Invalid passphrase`);
                } else {
                    const floating_div = document.createElement("div");
                    floating_div.classList = "form-floating";

                    const text_area = document.createElement("textarea");
                    text_area.classList = "form-control";
                    text_area.style = "height: 300px";
                    text_area.readOnly = true;
                    text_area.value = msg;

                    const lbl = document.createElement("label");
                    lbl.appendChild(document.createTextNode("Note Content"))

                    floating_div.appendChild(text_area);
                    floating_div.appendChild(lbl);
                    result_body.append(floating_div);
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
            result_body.text(`❌ Error: ${result.message}`);
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