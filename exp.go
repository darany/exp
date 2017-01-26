package main

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/aztec"
	"github.com/boombuler/barcode/codabar"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/qr"
	"github.com/boombuler/barcode/twooffive"
	"github.com/kardianos/osext"
	//TODO negroni, logger, rotate logs
)

func main() {

	ef, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatalf("ExecutableFolder failed: %v", err)
	}

	foldername := filepath.Join(ef, "static")
	log.Println("Static folder:", foldername)

	http.HandleFunc("/barcode.jpg", getBarcode)             // set router
	http.Handle("/", http.FileServer(http.Dir(foldername))) //File server
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	http.Serve(listener, nil)
}

//os.Hostname()
// Ip or base name from config file
//http://localhost:9090/barcode?message=benjamin&scale=a
func getBarcode(w http.ResponseWriter, r *http.Request) {
	message := "http://" + GetOutboundIP() + ":9090/"
	err := GenerateBarcode(w, "image/jpeg", message, "qr", "100")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Println(err)
	}
}

// GetOutboundIP gets preferred outbound ip of this machine
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")

	return localAddr[0:idx]
}

// GenerateBarcode generates a barcode and is generally called from a web context
// 'w' is the writer where the png ou jpeg image is written
// 'mimeType' is the requested datatype (e.g. image/jpeg)
// 'message' message contained into the codebar
// 'code' one of the supported barcode types (see github.com/boombuler/barcode)
// 'scale' string representation of scale percentage
func GenerateBarcode(w io.Writer, mimeType string, message string, code string, scale string) error {
	var err error
	scaleInt := 100
	messageByte := []byte(message)
	var codeOutput barcode.Barcode
	if scale != "" {
		scaleInt, err = strconv.Atoi(scale)
		if err != nil {
			scaleInt = 100
		}
	}
	if scaleInt < 0 {
		scaleInt = -scaleInt
	}

	//Default to qrcode previously built and used in 99% of cases
	switch code {
	case "aztec":
		codeOutput, err = aztec.Encode(messageByte, 33, 0)
	case "codabar":
		codeOutput, err = codabar.Encode(message)
	case "code128":
		codeOutput, err = code128.Encode(message)
	case "code39":
		codeOutput, err = code39.Encode(message, false, isASCII(message))
	case "datamatrix":
		codeOutput, err = datamatrix.Encode(message)
	case "ean":
		codeOutput, err = ean.Encode(message)
	case "qr":
		codeOutput, err = qr.Encode(message, qr.L, qr.Auto)
	case "twooffive":
		messageAndCheckSum, err2 := twooffive.AddCheckSum(message)
		if err2 == nil {
			codeOutput, err = twooffive.Encode(messageAndCheckSum, true)
		} else {
			err = err2
		}
	default:
		codeOutput, err = qr.Encode(message, qr.L, qr.Auto)
	}

	if err == nil {
		rect := codeOutput.Bounds()
		if rect.Dx() > 123 || rect.Dy() > 1 {
			codeOutput, err = barcode.Scale(codeOutput, scaleInt, scaleInt)
		}
		//codeOutput, err = barcode.Scale(codeOutput, scaleInt, scaleInt)
		if err != nil {
			log.Fatalf("GenerateBarcode: %v", err)
		} else {
			switch mimeType {
			case "image/jpeg":
				err = jpeg.Encode(w, codeOutput, nil)
			case "image/png":
				err = png.Encode(w, codeOutput)
			default:
				err = jpeg.Encode(w, codeOutput, nil)
			}
		}
	}

	if err != nil {
		log.Fatalf("GenerateBarcode: %v", err)
	}
	return err
}

// isASCII checks if 's' contains only ASCII characters
func isASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}
