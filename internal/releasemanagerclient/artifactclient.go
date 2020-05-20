package releasemanagerclient

import (
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/tracing"
)

type Client struct {
	ReleaseManagerURL       string
	ReleaseManagerAuthToken string
	Tracer                  tracing.Tracer
}

func (c *Client) PushArtifact(artifactSpec artifact.Spec, files []string) {

	// fileReader, err := ioutil.ReadFile(file)

	// md5s := base64.StdEncoding.EncodeToString(h.Sum(nil))
	// resp.HTTPRequest.Header.Set("Content-MD5", md5s)

	// req, err := http.NewRequest("PUT", url, fileReader)
	// req.Header.Set("Content-MD5", md5s)
	// if err != nil {
	// 		fmt.Println("error creating request", url)
	// 		return
	// }

	// defClient, err := http.DefaultClient.Do(req)
	// fmt.Println(defClient, err)
}

func (c *Client) zipFiles(files []string) {
	// // Create a buffer to write our archive to.
	// buf := new(bytes.Buffer)

	// // Create a new zip archive.
	// w := zip.NewWriter(buf)

	// // Add some files to the archive.
	// var files = []struct {
	// 	Name, Body string
	// }{
	// 	{"readme.txt", "This archive contains some text files."},
	// 	{"gopher.txt", "Gopher names:\nGeorge\nGeoffrey\nGonzo"},
	// 	{"todo.txt", "Get animal handling licence.\nWrite more examples."},
	// }
	// for _, file := range files {
	// 	f, err := w.Create(file.Name)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	_, err = f.Write([]byte(file.Body))
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	// // Make sure to check the error on Close.
	// err := w.Close()
	// if err != nil {
	// 	log.Fatal(err)
	// }
}
