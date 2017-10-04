class App {
    get imports() {
        return ['menu', 'router']
    }

    template() {
        return `<div class="site">
    <header class="main-header card-1">
        <h2 class="logo">Title</h2>
    </header>

    <div class="-menu header-menu"></div>

    <main class="main-content">
        <div class="-router content card card-1"></div>
    </main>
</div>`
    }

    templateLogin() {
        return `<div class="site">
    <header class="main-header card-1">
        <h2 class="logo">Login</h2>
    </header>

    <main class="main-content">
        <div class="-router"></div>
    </main>
</div>`
    }

    render(nodeElement, dataObject) {
        nodeElement.innerHTML = this.template();
    }
}

customComponents.define('app', App);
