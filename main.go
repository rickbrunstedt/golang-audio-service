package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

func greetingsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func serveIndexHtml(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func servePublic(w http.ResponseWriter, r *http.Request) {
	dir := http.Dir("./public")
	fileServer := http.FileServer(dir)
	fileServer = http.StripPrefix("/public/", fileServer)
	fileServer.ServeHTTP(w, r)
}

var audioStream io.Reader
var audioStreamLock sync.Mutex
var mic = exec.Command("ffmpeg", "-f", "alsa", "-i", "default", "-f", "mp3", "pipe:1")
var micAndOther = exec.Command("ffmpeg",
	"-f", "pulse", "-i", "alsa_output.pci-0000_01_00.1.hdmi-stereo.monitor",
	"-f", "pulse", "-i", "alsa_output.usb-Logitech_PRO_X_Wireless_Gaming_Headset-00.analog-stereo.monitor",
	"-filter_complex", "[0:a][1:a]amerge=inputs=2[aout]",
	"-map", "[aout]", "-f", "mp3", "pipe:1")

func startFFMpeg() {
	cmd := mic
	r, w := io.Pipe()
	cmd.Stdout = w
	audioStream = r

	// start command
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	// Close the writer when the command exists
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("Command finished with error: %v", err)
		}
		w.Close()
	}()
}

func streamAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "audio/mpeg")

	audioStreamLock.Lock()
	defer audioStreamLock.Unlock()

	if _, err := io.Copy(w, audioStream); err != nil {
		log.Printf("Error copying audio stream: %v", err)
	}
}

func main() {
	startFFMpeg()

	r := http.NewServeMux()
	r.HandleFunc("/", serveIndexHtml)
	r.HandleFunc("/hello", greetingsHandler)
	r.HandleFunc("/public/", servePublic)
	r.HandleFunc("/audio", streamAudio)

	http.Handle("/", r)

	println("Server started on port http://localhost:3000")
	http.ListenAndServe(":3000", nil)
}

func chk(err error) {
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
}
