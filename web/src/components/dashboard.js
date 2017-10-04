class Dashboard {

    template() {
        return `<h1>Dashboard</h1>`
    }

    render(nodeElement) {
        nodeElement.innerHTML = this.template()
    }
}

customComponents.define('dashboard', Dashboard);
