class Login {

    template() {
        return `
<style>
.login-window {
display: flex;
align-items: center;
flex-direction: column;
padding: 24px 48px 0;
}
.login-window > * {
width: 250px;
margin: 16px 0;
}
.login-window > .title {
font-size: 20px;
margin: 8px 0 24px;
}
.login-window > input[type=submit] {
align-self: flex-end;
}
</style>
<form class="js-form-login login-window">
<span class="title">Login to continue</span>
<input type="email" name="email" placeholder="Email" required>
<input type="password" name="password" placeholder="Password" required>
<input type="submit" value="Login">
</form>`
    }

    render(nodeElement) {
        nodeElement.innerHTML = this.template();

        let form = new Form(nodeElement, '.js-form-login', {
            loadingText: 'Logging you in',
            doneText: 'Login',
            errorSelector: '.js-error',
            defaultErrorMessage: 'There was an error submitting form'
        });

        form.onSubmit(function (formData, done) {
            App.api.post('auth/login', formData)
                .then(function (t) {
                    console.log(t);
                    App.profile = t.data.result;
                    App.renew(t.data);
                    done(true)
                })
                .catch(function (p1) {
                    console.log(p1);
                    done(false, 'Error')
                })
        });
    }
}

customComponents.define('login', Login);
