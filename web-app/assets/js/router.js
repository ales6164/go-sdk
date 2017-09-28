(function () {
    'use strict';

    window['Router'] = new function () {
        this.routerInstance = "app";
        this.selector = "#app";
        this.views = {};
        this.view = null;
        this.listeners = {};
        this.halter = {
            halt: true,
            triggers: []
        };

        this.navigate = function (view, path) {
            if (window.location.pathname !== path) window.history.pushState(
                {
                    name: this.routerInstance,
                    view: view,
                    path: path
                },
                "Title",
                path
            );

            this.load(view)
        };

        this.reload = function () {
            this.load(this.view);
        };

        this.load = function (view) {
            this.view = view;
            if (this.views.hasOwnProperty(view)) {
                this.render(view)
            } else {
                (function (c) {
                    $.ajax({
                        url: "/part/" + view + ".html",
                        context: {router: c, view: view},
                        type: "GET",
                        dataType: "html",
                        success: function (data) {
                            this.router.views[this.view] = data;
                            this.router.notify(this.view, "create", data, true);
                            this.router.render(this.view);
                        },
                        error: function (xhr, status) {
                            console.error("Error fetching html view", xhr, status);
                        },
                        complete: function () {
                            $('html, body').animate({scrollTop: '0px'}, 150);
                        }
                    });
                })(this)
            }
        };

        this.findLinks = function (parent) {
            (function (c) {
                $(parent).find("a").each(function () {
                    if($(this).attr("data-load")) {
                        $(this).on("click", function (e) {
                            var page = $(this).attr("data-load");
                            if (page) {
                                e.preventDefault();
                                c.navigate(page, $(this).attr("href"))
                            }
                        })
                    }
                });
            })(this)
        };

        this.render = function (view) {
            var html = this.views[view];
            this.notify($(this.selector).attr("data-view"), "unload");
            $(this.selector).attr("data-view", view);
            $(this.selector).html(html);
            this.findLinks(this.selector);
            this.notify(view, "load");
        };

        this.resolve = function () {
            this.view = $(this.selector).attr("data-view");
            if (this.view) {
                this.notify(this.view, "create");
                this.notify(this.view, "load");
            }
        };

        this.subscribe = function (type, obj) {
            if (!this.listeners.hasOwnProperty(type)) {
                this.listeners[type] = {};
            }

            for (var event in obj) {
                if (obj.hasOwnProperty(event)) {
                    var f = obj[event];
                    if (typeof f !== 'function') {
                        throw console.error("(" + type + ") " + event + ": " + f + "is not a function")
                    }

                    if (!this.listeners[type].hasOwnProperty(event)) {
                        this.listeners[type][event] = {};
                        this.listeners[type][event].todo = [];
                        this.listeners[type][event].data = [];
                    }
                    var i = 0;
                    if (this.listeners[type][event].ready) {
                        for (i; i < this.listeners[type][event].data.length; i++) {
                            f(this.listeners[type][event].data[i])
                        }
                    }
                    this.listeners[type][event].todo.push({f: f, i: i});
                }
            }

            return this.listeners[type][event].todo.length - 1;
        };

        this.unsubscribe = function (type, event, index) {
            if (!this.listeners.hasOwnProperty(type)) {
                return
            }
            if (!this.listeners[type].hasOwnProperty(event)) {
                return
            }
            this.listeners[type][event].todo.splice(index, 1);
        };

        this.unhalt = function () {
            this.halter.halt = false;
            this.halter.triggers.forEach(function (t) {
                t.context.notify(t.type, t.event, t.o, t.onlyFirsTime)
            })
        };

        this.notify = function (type, event, o, onlyFirsTime) {
            if (this.halter.halt) {
                this.halter.triggers.push({context: this, type: type, event: event, o: o, onlyFirsTime: onlyFirsTime});
                return
            }
            if (!this.listeners.hasOwnProperty(type)) {
                this.listeners[type] = {};
            }
            if (!this.listeners[type].hasOwnProperty(event)) {
                this.listeners[type][event] = {};
                this.listeners[type][event].todo = [];
                this.listeners[type][event].data = [];
            }
            this.listeners[type][event].data.push(o);
            this.listeners[type][event].ready = true;

            var listeners = this.listeners[type][event].todo;
            var arr = this.listeners[type][event].data;

            if (listeners) {
                for (var i = 0; i < listeners.length; i++) {
                    var obj = listeners[i];
                    if (onlyFirsTime && obj.i !== 0) continue;
                    if (typeof obj.f === 'function') {
                        for (var j = obj.i; j < arr.length; j++) {
                            obj.f.call(this.listeners[type], arr[j]);
                            obj.i += 1;
                        }
                    } else {
                        console.log(listeners[i], "is not a function")
                    }
                }
            }
        }
    };
})();