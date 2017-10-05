const gulp = require('gulp');
const babel = require('gulp-babel');
const copy = require('gulp-copy');
const sourcemaps = require('gulp-sourcemaps');
const browserSync = require('browser-sync').create();
const historyFallback = require('connect-history-api-fallback');
const reload = browserSync.reload;
const rename = require('gulp-rename');
const sass = require('gulp-sass');
const minifyCss = require('gulp-clean-css');

gulp.task('babel', function () {
    return gulp.src('src/components/*.js')
        .pipe(babel())
        .pipe(gulp.dest('dist/components'));
});

gulp.task('sass', function () {
    return gulp.src('./src/assets/css/*.scss')
        /*.pipe(sourcemaps.init())*/
        .pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError))
        /*.pipe(sourcemaps.write('./'))*/
        .pipe(gulp.dest('dist/assets/css'))
        .pipe(minifyCss({                             // minify CSS
            keepSpecialComments: 0                    // remove all comments
        }))
        .pipe(rename({                                // rename file
            suffix: ".min"                            // add *.min suffix
        }))
        .pipe(gulp.dest('dist/assets/css'));
});


gulp.task('copy', function () {
    return gulp.src([
        'src/*',
        'src/assets/js/**/*'
    ])
        .pipe(copy('dist', {prefix: 1}))
});

gulp.task('default', function () {

});

gulp.task('serve', ['sass', 'babel', 'copy'], function () {
    browserSync.init({
        port: 3000,
        server: {
            baseDir: 'dist',
            middleware: [
                historyFallback()
            ]
        }
    });

    gulp.watch(['src/*'], ['copy', reload]);
    gulp.watch(['src/components/*.js'], ['babel', reload]);
    gulp.watch(['src/assets/css/**/*'], ['sass', reload]);
});