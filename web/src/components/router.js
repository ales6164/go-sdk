/**
 * Dependencies: axios.js, navigo.js
 */
class Router {
    constructor() {
        (function (ctx) {
            ctx.navigo = new Navigo(null, false);
            ctx.navigo.on({
                'dashboard': function () {
                    ctx.renderPage('dashboard');
                },
                'entities/:kind': function (params) {
                    ctx.renderPage('entityListView', params.kind);
                },
                'entities': function (params) {
                    ctx.renderPage('entities');
                },
                '': function (params) {
                    ctx.navigo.navigate('/dashboard')
                },
                '*': function (params) {
                    ctx.renderPage('notFound');
                }
            });

            ctx.navigo.hooks({
                after: function (done, params) {
                    ctx.push('after', params)
                }
            });
        })(this);

        // axios instance
        this.ajax = axios.create({
            baseURL: '/components/',
            timeout: 1000,
        });
    }

    render() {
        this.navigo.resolve();
    }

    renderPage(componentName, data) {
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
}

customComponents.define('router', Router);
