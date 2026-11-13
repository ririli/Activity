// k8skeys initializes one isolated, internally consistent learning key set.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: k8skeys <directory>")
	}
	dir := os.Args[1]
	if err := os.MkdirAll(dir, 0700); err != nil {
		panic(err)
	}
	names := []string{"access.seed", "access.pub", "refresh.seed", "refresh.pub"}
	existing := 0
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			existing++
		}
	}
	if existing == len(names) {
		return
	}
	if existing != 0 {
		panic("partial key set: preserve existing files and investigate before retrying")
	}
	for _, kind := range []string{"access", "refresh"} {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		for name, data := range map[string][]byte{kind + ".pub": pub, kind + ".seed": priv.Seed()} {
			f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}
			_, err = fmt.Fprint(f, hex.EncodeToString(data))
			closeErr := f.Close()
			if err != nil {
				panic(err)
			}
			if closeErr != nil {
				panic(closeErr)
			}
		}
	}
}
