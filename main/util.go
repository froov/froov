package froov

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/pkg/browser"
)

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
func exists(f string) bool {
	_, err := os.Stat(f)
	return err == nil
}
func hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])[0:16]
}

func serve(root string) {
	// note that index.html is not at the root - should we
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, root+"/"+r.URL.Path[1:])
	})
	browser.OpenURL("http://localhost:8092")
	log.Fatal(http.ListenAndServe(":8092", nil))
}

func stem(p string) string {
	base := path.Base(p)
	ext := path.Ext(p)
	return base[:len(base)-len(ext)]
}
