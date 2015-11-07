package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

var client = http.Client{}

type resource struct {
	url         string
	data        []byte
	size        int64
	sectionSize int64
	sections    []section
	fileName    string
}

type section struct {
	id    int
	start int64
	end   int64
	data  []byte
}

func main() {

	d := &resource{
		url: "http://mirrors.mit.edu/pub/OpenBSD/doc/obsd-faq.txt",
	}

	req, err := http.NewRequest("HEAD", d.url, nil)
	if err != nil {
		fmt.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	d.size = resp.ContentLength
	d.sectionSize = d.size / 5
	d.data = make([]byte, d.size)

	ch := make(chan int)

	var j int64 = 0
	d.sections = make([]section, 5)
	for i := 0; i < 5; i++ {
		d.sections[i] = section{
			id:    i,
			data:  d.data[j : j+d.sectionSize],
			start: j,
		}
		j += d.sectionSize
		d.sections[i].end = j - 1
	}

	for _, s := range d.sections {
		s := s
		go download(&s, d.url, ch)
	}

	for i := 0; i < 5; i++ {
		<-ch
	}

	ioutil.WriteFile("file", d.data, os.ModePerm)
}

func download(s *section, url string, ch chan int) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Range", "bytes="+strconv.FormatInt(s.start, 10)+"-"+strconv.FormatInt(s.end, 10))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)

	var n int64

	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for _ = range ticker.C {
			fmt.Println("Section: " + strconv.Itoa(s.id) + "; speed: " + strconv.FormatInt(n/(1024*5), 10))
			n = 0
		}
	}()

	for {
		tn, err := r.Read(s.data)
		n = n + int64(tn)
		if err == io.EOF {
			fmt.Println(err)
			ticker.Stop()
			break
		}
	}

	fmt.Println("Section " + strconv.Itoa(s.id) + " completed")

	ch <- 0
}
