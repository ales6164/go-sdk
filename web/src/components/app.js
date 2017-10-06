/**
 * Dependencies: axios.js
 */
class App {
    get imports() {
        return ['menu', 'router']
    }

    template() {
        return `<header class="main-header card-1">
        <h2 class="logo">Title</h2>
    </header>

    <div class="-menu header-menu"></div>

    <main class="main-content">
        <div class="-router content card card-1"></div>
    </main>`
    }

    templateLogin() {
        return `
    <header class="main-header card-1">
        <h2 class="logo">Login</h2>
    </header>

    <main class="main-content">
        <div class="-router content-fluid card card-1"></div>
    </main>`
    }

    render(nodeElement, dataObject) {
        (function (ctx) {
            ctx.authenticateWithToken(function (profile) {
                if (profile) {
                    nodeElement.innerHTML = ctx.template();
                    console.log("pushing");
                    ctx.push('auth', profile);
                } else {
                    nodeElement.innerHTML = ctx.templateLogin();
                    ctx.push('auth', false);
                }
                ctx.renderImports(dataObject)
            })
        })(this)
    }

    /**
     * User management
     */
    authenticateWithToken(cb) {
        let token = App.token;

        if (token) {
            App.api.get('auth', {
                headers: {'Authorization': 'Bearer ' + token}
            }).then(function (t) {
                App.renew(t.data);
                if (t.data.result) {
                    cb(t.data.result);
                    return
                }
                App.token = null;
                App.profile = null;
                cb(false)
            }).catch(function (p1) {
                console.log(p1);
                App.token = null;
                App.profile = null;
                cb(false)
            })
        } else {
            console.log('Not authenticated');
            cb(false)
        }
    }

    static get token() {
        return localStorage.getItem('id_token')
    }

    static set token(token) {
        if (token && token.id && token.expires) {
            localStorage.setItem('id_token', token.id);
            localStorage.setItem('token_exp', token.expires)
        } else if (localStorage.getItem('id_token')) {
            localStorage.removeItem('id_token')
        }
    }

    static get profile() {
        return App._profile
    }

    static set profile(profile) {
        App._profile = profile
    }

    static renew(r) {
        if (r.token) {
            App.token = r.token;
        }
    }

    /**
     * API functions
     */

    static get api() {
        let headers = {};
        let token = App.token;
        if (token) {
            headers['Authorization'] = 'Bearer ' + token;
        }

        let baseURL = App.isLocalhost ? '//localhost:8080/api/' : '/api/';

        return axios.create({
            baseURL: baseURL,
            timeout: 20000,
            headers: headers
        });
    }

    static get isLocalhost() {
        return Boolean(window.location.hostname === 'localhost' ||
            // [::1] is the IPv6 localhost address.
            window.location.hostname === '[::1]' ||
            // 127.0.0.1/8 is considered localhost for IPv4.
            window.location.hostname.match(
                /^127(?:\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}$/
            )
        )
    }
}

customComponents.define('app', App);
