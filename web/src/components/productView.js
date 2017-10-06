class ProductView {

    template(url, fields, data) {
        const header = `<style>
.table-search {
padding: 16px;
display: flex;
}
</style>
<form class="js-form-search table-search">
<input type="text" placeholder="Search query" name="q">
<input type="submit" value="Search">
</form>
`;

        let tmpl;
        if (data) {
            tmpl = (url, fields, data) => `
${header}

    <table class="pretty-table">
    <tr><th>#</th>${fields.map(item => `
<th>${item}</th>
`).join('')}<th><!-- edit --></th>
</tr>
    ${data.map((item, i) => `
        <tr><td>${i + 1}</td>
        ${fields.map(name => `<td>${item[name]}</td>`).join('')}
        <td><a href="${url + item.id}" data-navigo><i class="fa fa-ellipsis-v"></i></a></td>
</tr>
       
    `).join('')}
    </table>
`;
        } else {
            tmpl = (url, fields, data) => `
${header}

    <table class="pretty-table">
    <tr><th>#</th>${fields.map(item => `
<th>${item}</th>
`).join('')}<th><!-- edit --></th>
</tr>
    <span>No entries</span>
</tr>
      </table>`;
        }

        return tmpl(url, fields, data)
    }

    render(nodeElement, dataKind) {
        //nodeElement.innerHTML = this.template(data)
        (function (ctx) {
            App.api.get(`entities/${dataKind}`)
                .then(function (t) {
                    if (t.data.result) {
                        ctx.updateResults(nodeElement, dataKind, t.data.result)
                    }
                })
                .catch(function (p1) {
                    console.error(p1);
                })
        })(this)
    }

    updateResults(nodeElement, dataKind, data) {
        console.log(data);

        nodeElement.innerHTML = this.template(`entities/${dataKind}/`, data.fields, data.data);

        (function (ctx) {
            let form = new Form(nodeElement, '.js-form-search', {formDataToMap: true});
            form.onSubmit(function (formData, done) {

                App.api.get(`entities/${dataKind}`, {
                    params: formData
                })
                    .then(function (t) {
                        console.log(t);
                        if (t.data.result) {
                            ctx.updateResults(nodeElement, dataKind, t.data.result);
                            done(true)
                        } else {
                            done(false)
                        }
                    })
                    .catch(function (p1) {
                        done(false, p1)
                    })
            });
        })(this);

        Router.navigo.updatePageLinks()
    }

}

customComponents.define('productView', ProductView);
