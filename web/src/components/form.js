class Form {
    constructor(node, selector, config) {
        this.node = node;
        this.selector = selector;
        this.form = node.querySelector(selector);
        this.config = config || {};
        this.submitBtn = this.form.querySelector('[type=submit]');

        if (!this.submitBtn) {
            console.error('Form has no submit button defined');
            return
        }

        if (!this.config.doneText) {
            this.config.doneText = this.btnText
        }

        if (!this.config.loadingText) {
            this.config.loadingText = this.btnText
        }
    }

    onSubmit(cb) {
        let done = function (ctx) {
            return function (successful, message) {
                ctx.btnText = ctx.config.doneText;
                ctx.submitBtn.disabled = false;

                if (successful) {
                    ctx.error = null;
                    ctx.success = message ? message : null
                } else {
                    ctx.error = message ? message : ctx.config.defaultErrorMessage;
                }
            }
        };
        (function (ctx) {
            ctx.listener = function (e) {
                e.preventDefault();
                ctx.submitBtn.disabled = true;
                ctx.btnText = ctx.config.loadingText;

                let fd = new FormData(this);

                if (ctx.config.formDataToMap) {
                    let result = {};
                    for (let entry of fd.entries()) {
                        result[entry[0]] = entry[1];
                    }
                    cb(result, done(ctx))
                } else {
                    cb(fd, done(ctx))
                }
            };

            ctx.form.addEventListener('submit', ctx.listener)
        })(this)
    }

    get btnText() {
        if (this.submitBtn.nodeName === "INPUT") {
            return this.submitBtn.value
        } else {
            return this.submitBtn.textContent
        }
    }

    set btnText(value) {
        if (this.submitBtn.nodeName === "INPUT") {
            this.submitBtn.value = value
        } else {
            this.submitBtn.textContent = value
        }
    }

    set error(value) {
        if (this.config.errorSelector) {
            let errElement = this.node.querySelector(this.config.errorSelector);
            if (errElement) {
                errElement.textContent = value
            }
        }
        if (value) {
            console.log("form" + this.selector + " error ", value)
        }
    }

    set success(value) {
        if (this.config.successSelector) {
            let sucElement = this.node.querySelector(this.config.successSelector);
            if (sucElement) {
                sucElement.textContent = value
            }
        }
        if (value) {
            console.log("form" + this.selector + " ", value)
        }
    }
}
