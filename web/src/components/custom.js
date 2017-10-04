const customComponents = new function () {
    this.definedComponents = new Map();

    this.define = function (name, componentClass) {
        console.log('defined ', name);
        this.definedComponents.set(name, componentClass)
    };

    this.hasComponent = function (name) {
        return this.definedComponents.has(name)
    };

    this.renderComponent = function (name, node, dataObject) {
        (function (ctx) {
            let componentClass = ctx.definedComponents.get(name);

            if(componentClass) {
                componentClass.prototype.element = node;
                componentClass.prototype.push = function (event, data, onlyOnce) {
                    ctx.eventHolder.notify(name, event, data, onlyOnce)
                };

                let component = new componentClass();

                component.render(node, dataObject);
                ctx.render(component.imports, node, dataObject);

                if (component.onInit) {
                    component.onInit.call(component)
                }

                ctx.registerListeners(name, component, component.listeners);
            } else {
                console.log('Component ' + name + ' is not defined')
            }
        })(this);
    };

    this.render = function (importArray, nodeElement, dataObject) {
        (function (ctx) {
            if (importArray) {
                importArray.forEach(function (name) {
                    let query = nodeElement.querySelectorAll('.-' + name);
                    for (let i = 0; i < query.length; i++) {
                        let node = query.item(i);

                        ctx.renderComponent(name, node, dataObject);
                    }
                })
            }
        })(this);
    };

    this.registerListeners = function (componentName, componentContext, listeners) {
        if (listeners) {
            for (let listenToComponent in listeners) {
                if (listeners.hasOwnProperty(listenToComponent)) {
                    this.eventHolder.subscribe(componentContext, listenToComponent, listeners[listenToComponent])
                }
            }
        }
    };

    this.eventHolder = {
        listeners: {},
        halter: {
            halt: true,
            triggers: []
        },
        subscribe: function (ctx, type, obj) {
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
                    this.listeners[type][event].todo.push({f: f, i: i, c: ctx});
                }
            }

            return this.listeners[type][event].todo.length - 1;
        },
        unsubscribe: function (type, event, index) {
            if (!this.listeners.hasOwnProperty(type)) {
                return
            }
            if (!this.listeners[type].hasOwnProperty(event)) {
                return
            }
            this.listeners[type][event].todo.splice(index, 1);
        },
        unhalt: function () {
            this.halter.halt = false;
            this.halter.triggers.forEach(function (t) {
                t.context.notify(t.type, t.event, t.o, t.onlyFirsTime)
            })
        },
        notify: function (type, event, o, onlyFirsTime) {
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
                            obj.f.call(obj.c, arr[j]);
                            obj.i += 1;
                        }
                    } else {
                        console.log(listeners[i], "is not a function")
                    }
                }
            }
        }
    };
};