/**
 * Dependencies: axios.js, navigo.js
 */
class Router {
    get listeners() {
        return {
            app: {
                auth: this.onAppAuth
            }
        }
    }

    constructor() {
        (function (ctx) {
            Router.navigo = new Navigo(null, false);
            Router.navigo.on({
                'dashboard': function () {
                    ctx.renderPage('dashboard', true);
                },
                'entities/:kind/:id': function (params) {
                    //ctx.renderPage('entityListView', true, params.kind);
                },
                'entities/:kind': function (params) {
                    ctx.renderPage('entityListView', true, params.kind);
                },
                'entities': function (params) {
                    ctx.renderPage('entities', true);
                },
                '': function (params) {
                    Router.navigo.navigate('/dashboard', true)
                },
                '*': function (params) {
                    ctx.renderPage('notFound');
                }
            });

            Router.navigo.hooks({
                after: function (done, params) {
                    ctx.push('after', params)
                }
            });
        })(this);

        // axios instance
        this.ajax = axios.create({
            baseURL: '/components/',
            timeout: 5000
        });
    }

    render() {}

    renderPage(componentName, needsAuthorization, data) {
        if (needsAuthorization && !this.profile) {
            componentName = 'login';
        }

        (function (ctx) {
            if (customComponents.hasComponent(componentName)) {
                customComponents.renderComponent(componentName, ctx.element, data);
            } else {
                ctx.ajax.get(componentName + '.js')
                    .then(function (response) {
                        let script = document.createElement("script");
                        script.type = 'text/javascript';
                        script.async = true;
                        script.innerHTML = response.data;
                        document.body.appendChild(script);

                        customComponents.renderComponent(componentName, ctx.element, data);

                        script.outerHTML = ""
                    })
                    .catch(function (error) {
                        console.log(error);
                    });
            }
        })(this);
    }

    onAppAuth(profile) {
        console.log("listened");

        this.profile = profile;
        Router.navigo.resolve();
    }
}

customComponents.define('router', Router);
