package openspeech

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func gzipCompress(input []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(input)
	w.Close()
	return b.Bytes()
}

func gzipDecompress(input []byte) []byte {
	b := bytes.NewBuffer(input)
	r, _ := gzip.NewReader(b)
	out, _ := ioutil.ReadAll(r)
	r.Close()
	return out
}

func LogJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}
