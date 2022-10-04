package websocket

import (
	"archive/tar"
	"bytes"
	"embed"
	"io"
	"os"
	"path/filepath"

	"github.com/lyyyuna/gococo/pkg/log"
)

//go:embed websocket.tar
var depTarFile embed.FS

// AddCustomWebsocketDep injects custom gorrila/websocket library into the temporary directory
//
//  1. untar websocket.tar from the embed file system
//  2. the websocket library is has no other dependency, appropriate for injecting
func AddCustomWebsocketDep(customWebsocketPath string) {
	data, err := depTarFile.ReadFile("websocket.tar")
	if err != nil {
		log.Fatalf("cannot find the websocket.tar in the embed file: %v", err)
	}

	buf := bytes.NewBuffer(data)
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("cannot untar the websocket.tar: %v", err)
		}

		fpath := filepath.Join(customWebsocketPath, hdr.Name)
		if hdr.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, hdr.FileInfo().Mode())
			if err != nil {
				log.Fatalf("fail to untar the websocket.tar: %v", err)
			}
		} else {
			fdir := filepath.Dir(fpath)
			err := os.MkdirAll(fdir, hdr.FileInfo().Mode())
			if err != nil {
				log.Fatalf("fail to untar the websocket.tar: %v", err)
			}

			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				log.Fatalf("fail to untar the websocket.tar: %v", err)
			}
			defer f.Close()

			_, err = io.Copy(f, tr)

			if err != nil {
				log.Fatalf("fail to untar the websocket.tar: %v", err)
			}
		}
	}
}
