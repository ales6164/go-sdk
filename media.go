package sdk

import (
	"cloud.google.com/go/storage"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/image"
	"io/ioutil"
	"net/http"
	"path"
)

var bucketName string
var mediaDir string

func MediaUploadHandler(bucket string, dir string) http.Handler {
	bucketName = bucket
	mediaDir = dir
	return http.HandlerFunc(upload)
}

func upload(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r)

	if appengine.IsDevAppServer() {
		ctx.PrintError(w, errors.New("production server required"), http.StatusInternalServerError)
		return
	}

	p, err := saveFile(ctx, "file")
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, p.(string))
}

func saveImage(ctx Context, name string) (interface{}, error) {
	filePath, err := save(ctx, name)
	if err != nil {
		return filePath, err
	}

	gsPath := path.Join("/gs/", bucketName, filePath)

	blobKey, err := blobstore.BlobKeyForFile(ctx.Context, gsPath)
	if err != nil {
		return filePath, errors.New("error reading file '" + gsPath + "': " + err.Error())
	}

	// Resize and crop the srcImage to fill the 100x100px area.
	servingUrl, err := image.ServingURL(ctx.Context, blobKey, &image.ServingURLOptions{
		Secure: true,
	})
	if err != nil {
		return filePath, errors.New("error serving url: " + err.Error())
	}

	return servingUrl.String(), nil
}

func saveFile(ctx Context, name string) (interface{}, error) {
	p, err := save(ctx, name)
	if err != nil || len(p) == 0 {
		return p, err
	}

	return "https://storage.googleapis.com/" + bucketName + "/" + p, nil
}

func save(ctx Context, name string) (string, error) {
	var p string
	var err error

	fileMultipart, fileHeader, err := ctx.r.FormFile(name)
	if err != nil {
		fileMultipart, fileHeader, err = ctx.r.FormFile("file")
		if err != nil {
			return p, err
		}
	}
	defer fileMultipart.Close()

	fileKeyName := uuid.New().String()

	bytes, err := ioutil.ReadAll(fileMultipart)
	if err != nil {
		return p, errors.New("error reading uploaded file: " + err.Error())
	}

	return writeFile(ctx.Context, fileKeyName, fileHeader.Filename, bytes)
}

func writeFile(ctx context.Context, key string, name string, bs []byte) (string, error) {
	var p string
	var err error

	client, err := storage.NewClient(ctx)
	if err != nil {
		return p, err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)

	p = mediaDir + "/" + key + "--" + name

	obj := bucket.Object(p)
	wc := obj.NewWriter(ctx)

	_, err = wc.Write(bs)
	if err != nil {
		return p, err
	}
	err = wc.Close()
	if err != nil {
		return p, err
	}

	acl := obj.ACL()
	if err := acl.Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return p, err
	}

	return p, err
}
