package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/figment-networks/cosmos-indexer/worker/api/coda"
)

var address string
var codaPath string

func init() {
	flag.StringVar(&address, "address", "", "srv address")
	flag.StringVar(&codaPath, "coda_path", "./data/coda", "Coda path")
	flag.Parse()
}

func main() {

	cm := NewCodaMem()

	mux := http.NewServeMux()
	mux.HandleFunc("/coda", cm.ServeCoda)

	filepath.Walk(codaPath, cm.CodaWalkFunc)

	//	log.Printf("%+v", cm.Store)
	s := &http.Server{
		Addr:    address,
		Handler: mux,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	log.Printf("Running server on %s", address)
	log.Fatal(s.ListenAndServe())
}

type CodaMem struct {
	Store map[string]coda.Block

	dec json.Decoder
}

func NewCodaMem() *CodaMem {
	return &CodaMem{
		Store: make(map[string]coda.Block),
	}
}

func (cm *CodaMem) CodaWalkFunc(path string, info os.FileInfo, err error) error {

	f, err := os.Open(path)
	defer f.Close()
	b := &coda.Block{}
	dec := json.NewDecoder(f)
	err = dec.Decode(b)
	if err == nil {
		//	n := strings.Replace(info.Name(), ".json", "", 1)
		//	cm.Store[n] = *b

		cm.Store[b.StateHash] = *b

	}

	return nil
}

func (cm *CodaMem) ServeCoda(w http.ResponseWriter, r *http.Request) {

	a := map[string]interface{}{}
	dec := json.NewDecoder(r.Body)

	dec.Decode(&a)

	//hash := "D2rcXVQa7H1zPu2qZoGsnxLiHcZJdutSBVf6K57QdausbBeF76FhVCWc2VWpQw3ccy8vqREVaQvBsbPRrBL8NKVzh9RAxRe232EZw7x16YNaHuzh1WbbmnrjvGpJsxWZVTyxmuqhwfRXSAWLnWeMBJBf8F1ck3Jteo5eNDUHp7viMXonxGUceQHkeRNokAMYSpBkGZ617ngQgQk7VqP5CmQ6277mdwp4dKsKv7W7S662q6StF7eguFMbQXEtCogxH5KGXZ86ZfKrjFEE33jpZf7XEdNfvbpWg4KHKC8oDF2e39L5Nh8RFTwXB96eAABPmX"

	reg := regexp.MustCompile(`stateHash:\s+"([^"]+)"`)
	query, ok := a["query"]
	if !ok {
		log.Println("query not present")
		w.WriteHeader(http.StatusBadRequest)
		return

	}
	q, ok := query.(string)
	if !ok {
		log.Println("query is not string")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := reg.FindStringSubmatch(q)
	if len(hash) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("hash is ", hash)
	enc := json.NewEncoder(w)

	//var i = 0

	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(`{"errors":[],"data":`))

	enc.Encode(cm.Store[hash[1]])
	/*
		for _, v := range cm.Store {
			if i != 0 {
				w.Write([]byte(","))
			}
			enc.Encode(v)
			i++
		}*/

	w.Write([]byte("}"))
}
