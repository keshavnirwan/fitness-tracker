async function hashSHA256(message) {
    const encoder = new TextEncoder();
    const data = encoder.encode(message);
    const hashBuffer = await window.crypto.subtle.digest('SHA-256', data);
    return Array.from(new Uint8Array(hashBuffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
}

async function hashPasswordAndSubmit(event) {
    event.preventDefault();
    console.log("Hashing function called!"); // Add this line
    const form = event.target;
    const passwordInput = form.querySelector('input[name="password"]');
    if (passwordInput && passwordInput.value) {
        passwordInput.value = await hashSHA256(passwordInput.value);
    }
    form.submit();
}

 