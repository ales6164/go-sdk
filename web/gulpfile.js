const gulp = require('gulp');
const glob = require('glob');
const sourcemaps = require('gulp-sourcemaps');
const sass = require('gulp-sass');
const del = require('del');
const read = require('read-file');
const readData = require('read-data').json;
const write = require('write-file');
const copy = require('gulp-copy');
const mustache = require('mustache');
const browserSync = require('browser-sync').create();
const reload = browserSync.reload;

//const appHelper = require('app-helper');
const appHelper = require('/home/ales/WebstormProjects/npm-app');

const appDir = "./app/";

gulp.task('test', ['scss', 'copy'], function () {
    var app = new appHelper.AppLite();
    app.parse(appDir + "**/*.html");
    app.compile();

    var router = new appHelper.RouterLite(app, {
        writeDir: "./dist/",
        outlet: "#app"
    });

    app.write(app.views["index"], "dist/part/index.html");

    // Index and home
    router.layout("index")
        .partial("home")
        .write("index.html")
        .layout(null)
        .write("part/home.html");

    var add = function (partial) {
        router.layout("index")
            .partial(partial)
            .write(partial + "/index.html")
            .layout(null)
            .write("part/" + partial + ".html");
    };

    var component = function (partial) {
        router.partial(partial)
            .write("comp/" + partial + ".html");
    };


});

gulp.task('serve', ['test'], function () {
    browserSync.init({
        port: 3000,
        server: {
            baseDir: 'dist'/*,
            middleware: [
                {
                    route: "/products",
                    handle: function (req, res, next) {
                        res.setHeader('Content-Type', 'text/html; charset=UTF-8');
                        next();
                    }
                }
            ]*/
        }
    });

    gulp.watch([appDir + '**/*.html'], ['test', reload]);
    gulp.watch(staticFiles, ['copy', reload]);
    gulp.watch([appDir + 'styles/*.scss'], ['scss', reload]);
});

const staticFiles = [
    appDir + '*',
    appDir + 'styles/**/*.{css,map}',
    appDir + '**/*.{js,map}',
    appDir + 'images/**/*'
];

gulp.task('copy', function () {
    return gulp.src(staticFiles)
        .pipe(copy("./dist", {prefix: 1}))
});

gulp.task('copyToGo', function () {
    return gulp.src('./dist/**/*')
        .pipe(copy("/home/ales/go/src/tiskdaril/src/static", {prefix: 1}))
});

gulp.task('clean', function () {
    return del([
        'dist/**/*'
    ]);
});

gulp.task('scss', function () {
    return gulp.src(appDir + 'styles/*.scss')
        .pipe(sourcemaps.init())
        .pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError))
        .pipe(sourcemaps.write('./'))
        .pipe(gulp.dest('./dist/styles'));
});