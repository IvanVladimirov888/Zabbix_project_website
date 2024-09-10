

document.addEventListener('DOMContentLoaded', function () {
    const loginForm = document.getElementById('login-form');
    if (loginForm) {
        loginForm.addEventListener('submit', login);
    }

    function login(event) {
        event.preventDefault();

        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;

        fetch('/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        })
            .then(response => {
                if (response.redirected) {
                    window.location.href = "main.html";
                } else {
                    alert("Ошибка авторизации. Попробуйте снова.");
                }
            })
            .catch(error => {
                console.error("Ошибка:", error);
            });
    }
});
