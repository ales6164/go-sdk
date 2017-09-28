(function () {
    'use strict';

    Router.subscribe('home', {
        load: function () {

        }
    });

    // load on document ready
    $(function () {
        Router.unhalt();
        Router.resolve();

        Router.findLinks('body');
        Router.load('home')
    })
})();