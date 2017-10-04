class EntityListView {

    template(data) {
        return `
<p>Results for: ${data}</p>
        <table>
        <tr>
        <th></th>
        </tr>
        </table>
        `
    }

    render(nodeElement, data) {
        nodeElement.innerHTML = this.template(data)
    }
}

customComponents.define('entityListView', EntityListView);
