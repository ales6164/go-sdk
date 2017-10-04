class Menu {

    get listeners() {
        return {
            router: {
                after: this.onRouted
            }
        }
    }

    template() {
        return `
        <a href="#" role="button" class="menu-btn drop-next js-drop-next">
            <i class="fa fa-bars" aria-hidden="true"></i>
        </a>
        <nav class="menu drop card-2">
            <a href="/dashboard" data-navigo>Domov</a>
            <a href="#" class="drop-next js-drop-next">Entities</a>
            <div class="drop">
                <a href="#" class="drop-next js-drop-next"><i class="fa fa-tags"></i>Ostalo</a>
                <div class="drop">
                    <a href="#">Stran 1</a>
                    <a href="#">Naslov</a>
                    <a href="#">Več</a>
                </div>
                <a href="/entities/product" data-navigo>Product</a>
            </div>
            <a href="#">Kontakt</a>
        </nav>
    `
    }

    render(nodeElement, dataObject) {
        nodeElement.innerHTML = this.template();
    }

    onInit() {
        var jsDropNext = this.element.querySelectorAll('.js-drop-next');
        for (var i = 0; i < jsDropNext.length; i++) {
            var a = jsDropNext[i];
            a.addEventListener('click', function (e) {
                e.preventDefault();
                if (this.hasAttribute('expanded')) {
                    this.removeAttribute('expanded')
                } else {
                    this.setAttribute('expanded', '')
                }
            })
        }
    }

    // listener function
    onRouted() {
        var jsDropNext = this.element.querySelectorAll('.js-drop-next');
        for (var i = 0; i < jsDropNext.length; i++) {
            var a = jsDropNext[i];
            if (a.hasAttribute('expanded')) {
                a.removeAttribute('expanded')
            }
        }
    }
}

customComponents.define('menu', Menu);
