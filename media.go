package sdk

import (
	"net/http"
	"github.com/google/uuid"
	"io/ioutil"
	"google.golang.org/appengine"
	"errors"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/image"
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

	ctx.Print(w, "https://storage.googleapis.com/"+bucketName+"/"+p.(string))
}

func saveImage(ctx Context, name string) (interface{}, error) {
	filePath, err := saveFile(ctx, name)
	if err != nil {
		return filePath, err
	}

	gsPath := path.Join("/gs/", bucketName, filePath.(string))

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
	var p string
	var err error

	fileMultipart, fileHeader, err := ctx.r.FormFile(name)
	if err != nil {
		/*if ctx.r.MultipartForm == nil {
			return path, errors.New("multipart form is nil")
		}

		if ctx.r.MultipartForm.File == nil {
			return path, errors.New("multipart file is nil")
		}

		if fhs := ctx.r.MultipartForm.File[name]; len(fhs) > 0 {
			return path, errors.New("multipart file exists")
		} else {
			var otherFiles = ""
			for name := range ctx.r.MultipartForm.File {
				otherFiles += name + ", "
			}
			return path, errors.New("multipart file array is empty for field '" + name + "'; there might be other fields: " + otherFiles)
		}*/
		return p, err
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
