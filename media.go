package sdk

import (
	"net/http"
	"github.com/google/uuid"
	"io/ioutil"
	"google.golang.org/appengine"
	"errors"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
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
		ctx.PrintError(w, errors.New("Production server required"), http.StatusInternalServerError)
		return
	}

	path, err := saveFile(ctx, "file")
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, "https://storage.googleapis.com/" + bucketName + "/" + path)
}

func saveFile(ctx Context, name string) (string, error) {
	var path string
	var err error

	fileMultipart, fileHeader, err := ctx.r.FormFile(name)
	if err != nil {
		return path, err
	}
	defer fileMultipart.Close()

	fileKeyName := uuid.New().String()

	bytes, err := ioutil.ReadAll(fileMultipart)
	if err != nil {
		return path, err
	}

	return writeFile(ctx.Context, fileKeyName, fileHeader.Filename, bytes)
}

func writeFile(ctx context.Context, key string, name string, bs []byte) (string, error) {
	var path string
	var err error

	client, err := storage.NewClient(ctx)
	if err != nil {
		return path, err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)

	path = mediaDir + "/" + key + "--" + name

	obj := bucket.Object(path)
	wc := obj.NewWriter(ctx)

	_, err = wc.Write(bs)
	if err != nil {
		return path, err
	}
	err = wc.Close()
	if err != nil {
		return path, err
	}

	acl := obj.ACL()
	if err := acl.Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return path, err
	}

	return path, err
}