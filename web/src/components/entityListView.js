class EntityListView {

    template(url, fields, data) {

        const tmpl = (url, fields, data) => `
<style>
.table-search {

}
</style>
<div class="table-search"></div>
    <table class="pretty-table">
    <tr><th>#</th>${fields.map(item => `
<th>${item}</th>
`).join('')}<th><!-- edit --></th>
</tr>
    ${data.map((item, i) => `
        <tr><td>${i+1}</td>
        ${fields.map(name => `<td>${item[name]}</td>`).join('')}
        <td><a href="${url + item.id}" data-navigo><i class="fa fa-ellipsis-v"></i></a></td>
</tr>
       
    `).join('')}
    </table>
`;
        return tmpl(url, fields, data)
    }

    render(nodeElement, dataKind) {
        //nodeElement.innerHTML = this.template(data)
        (function (ctx) {
            App.api.get(`entities/${dataKind}`)
                .then(function (t) {
                    if(t.data.result) {
                        console.log(t.data.result);
                        nodeElement.innerHTML = ctx.template(`entities/${dataKind}/`, t.data.result.fields, t.data.result.data);
                        Router.navigo.updatePageLinks()
                    }
                })
                .catch(function (p1) {
                    console.error(p1)
                })
        })(this)
    }

}

customComponents.define('entityListView', EntityListView);
